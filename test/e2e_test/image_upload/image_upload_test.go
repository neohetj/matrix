package image_upload

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	matrix "github.com/neohetj/matrix"
	"github.com/neohetj/matrix/internal/registry"
	"github.com/neohetj/matrix/pkg/components/endpoint"
	"github.com/neohetj/matrix/pkg/config"
	"github.com/neohetj/matrix/test/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper struct for setup
type TestEnv struct {
	Engine     *matrix.MatrixEngine
	MockLogger *utils.MockLogger
}

// setup initializes the Matrix engine and mock logger for testing
func setup(t *testing.T) *TestEnv {
	cleanup()

	mockLogger := &utils.MockLogger{}
	cfg := config.MatrixConfig{
		Loader: config.LoaderConfig{
			Providers: []config.LoaderProviderConfig{
				{
					Type: "file",
					Args: []string{".."}, // Search from parent directory
				},
			},
			ComponentsRoot: ".",
		},
		EnabledComponents: []string{"image_upload"},
	}
	eng, err := matrix.New(cfg, matrix.WithLogger(mockLogger))
	assert.NoError(t, err)

	return &TestEnv{
		Engine:     eng,
		MockLogger: mockLogger,
	}
}

func cleanup() {
	registry.Default.RuntimePool.Unregister("rc-image-upload")
	registry.Default.SharedNodePool.Stop()
	os.RemoveAll("uploads") // Cleanup uploaded files
}

func TestImageUpload(t *testing.T) {
	defer cleanup()
	env := setup(t)

	// 1. Retrieve the loaded endpoint
	// Note: The endpoint ID is usually derived from filename or definition.
	// Since we haven't specified an ID in the JSON, it might default to filename "image_upload_endpoint"
	// or we need to check how loader assigns IDs.
	// Let's assume the loader uses the filename without extension as ID if not specified,
	// BUT config says "EndpointsPath" is scanned.
	// In the JSON I created: endpoints/image_upload_endpoint.json
	// The loader typically registers endpoints.
	// I'll check if I can find it by expected ID "ep_api_upload" (derived from path?) No.
	// The `http_endpoint.go` doesn't auto-generate ID from path.
	// The loader uses the filename as the key in the map passed to manager.
	// So ID should be "image_upload_endpoint".

	epCtx, ok := env.Engine.SharedNodePool().Get("image_upload_endpoint")
	require.True(t, ok, "endpoint 'image_upload_endpoint' not found")

	epNode, ok := epCtx.GetNode().(endpoint.HttpEndpoint)
	require.True(t, ok, "node is not an HttpEndpoint")

	// 2. Create Multipart Request
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("image", "test_image.png")
	assert.NoError(t, err)

	// Read real image file
	imagePath := utils.GetTestImagePath()
	fileContent, err := os.ReadFile(imagePath)
	if err != nil {
		t.Fatalf("Failed to read test image from %s: %v", imagePath, err)
	}

	_, err = io.Copy(part, bytes.NewReader(fileContent))
	assert.NoError(t, err)

	writer.Close()

	req := httptest.NewRequest("POST", "/api/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	// 3. Handle Request
	err = epNode.HandleHttpRequest(w, req)
	assert.NoError(t, err)

	// 4. Verify Response
	resp := w.Result()
	assert.Equal(t, 200, resp.StatusCode)

	respBody, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)

	type Response struct {
		Path string `json:"path"`
		Type string `json:"type"`
		Size int64  `json:"size"`
		MD5  string `json:"md5"`
	}
	var respData Response
	err = json.Unmarshal(respBody, &respData)
	assert.NoError(t, err)

	// 5. Verify File Saved
	// The SaveFile function defaults to "temp" directory if not configured.
	// The rule chain configuration sets "uploadRootDir" to "uploads".
	expectedPath := filepath.Join("uploads", "test_image.png")
	savedContent, err := os.ReadFile(expectedPath)
	assert.NoError(t, err)
	assert.Equal(t, fileContent, savedContent)

	// 6. Verify Metadata
	assert.Equal(t, expectedPath, respData.Path)
	assert.Equal(t, int64(len(fileContent)), respData.Size)

	hash := md5.Sum(fileContent)
	expectedMD5 := hex.EncodeToString(hash[:])
	assert.Equal(t, expectedMD5, respData.MD5)

	// 7. Verify Logs
	logOutput := env.MockLogger.String()
	assert.Contains(t, logOutput, "File uploaded and saved successfully")
}
