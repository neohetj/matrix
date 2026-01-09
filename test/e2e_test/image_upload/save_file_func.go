package image_upload

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"

	"github.com/neohetj/matrix/internal/registry"
	"github.com/neohetj/matrix/pkg/asset"
	"github.com/neohetj/matrix/pkg/cnst"
	"github.com/neohetj/matrix/pkg/helper"
	"github.com/neohetj/matrix/pkg/message"
	"github.com/neohetj/matrix/pkg/types"
)

const (
	SaveFileFuncID  = "SaveFile"
	FileObjectSID   = "MultipartFileHeader"
	FileMetadataSID = "FileMetadata"
)

type FileMetadata struct {
	Path string `json:"path"`
	Type string `json:"type"`
	Size int64  `json:"size"`
	MD5  string `json:"md5"`
}

func (f *FileMetadata) GetPath() string { return f.Path }

// SaveFile is a custom function to save content to a file.
// It expects a "file" parameter pointing to a CoreObj wrapping *multipart.FileHeader.
func SaveFile(ctx types.NodeCtx, msg types.RuleMsg) {
	assetCtx := asset.NewAssetContext(asset.WithNodeCtx(ctx), asset.WithRuleMsg(msg))

	// Get upload root dir from config or default to "temp"
	uploadRootDir, err := helper.GetConfigAsset[string](assetCtx, "uploadRootDir")
	if err != nil || uploadRootDir == "" {
		uploadRootDir = "temp"
	}

	// 1. Get the FileHeader using generic helper
	fileHeader, err := helper.GetParam[*multipart.FileHeader](assetCtx, "file")
	if err != nil {
		ctx.TellFailure(msg, types.InvalidParams.Wrap(err))
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		ctx.TellFailure(msg, types.InternalError.Wrap(fmt.Errorf("failed to open file header: %w", err)))
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		ctx.TellFailure(msg, types.InternalError.Wrap(fmt.Errorf("failed to read file content: %w", err)))
		return
	}

	// 4. Save file
	filename := fileHeader.Filename
	if filename == "" {
		ctx.TellFailure(msg, types.InvalidParams.Wrap(fmt.Errorf("filename is empty")))
		return
	}

	if err := os.MkdirAll(uploadRootDir, 0755); err != nil {
		ctx.TellFailure(msg, types.InternalError.Wrap(fmt.Errorf("failed to create directory: %w", err)))
		return
	}

	outputPath := filepath.Join(uploadRootDir, filename)
	if err := os.WriteFile(outputPath, content, 0644); err != nil {
		ctx.TellFailure(msg, types.InternalError.Wrap(fmt.Errorf("failed to write file: %w", err)))
		return
	}

	// Calculate MD5
	hash := md5.Sum(content)
	md5Str := hex.EncodeToString(hash[:])

	// Create metadata object
	meta := &FileMetadata{
		Path: outputPath,
		Type: fileHeader.Header.Get("Content-Type"),
		Size: fileHeader.Size,
		MD5:  md5Str,
	}

	// Return as CoreObj
	// Use NewItemByParam to create the object mapped to "result" parameter
	coreObj, err := msg.DataT().NewItemByParam(ctx, "result")
	if err != nil {
		ctx.TellFailure(msg, types.InternalError.Wrap(fmt.Errorf("failed to create result object: %w", err)))
		return
	}

	if err := coreObj.SetBody(meta); err != nil {
		ctx.TellFailure(msg, types.InternalError.Wrap(fmt.Errorf("failed to set result body: %w", err)))
		return
	}

	ctx.TellSuccess(msg)
}

func init() {
	// Register the MultipartFileHeader CoreObj definition
	registry.Default.CoreObjRegistry.Register(
		message.NewCoreObjDef(&multipart.FileHeader{}, FileObjectSID, "A multipart file header object"),
	)

	// Register the FileMetadata CoreObj definition
	registry.Default.CoreObjRegistry.Register(
		message.NewCoreObjDef(&FileMetadata{}, FileMetadataSID, "File metadata object"),
	)

	// Register the SaveFile function
	registry.Default.NodeFuncManager.Register(&types.NodeFuncObject{
		Func: SaveFile,
		FuncObject: types.FuncObject{
			ID:   SaveFileFuncID,
			Name: "Save File",
			Desc: "Saves a MultipartFileHeader to disk",
			Configuration: types.FuncObjConfiguration{
				Inputs: []types.IOObject{
					{ParamName: "file", DefineSID: FileObjectSID},
				},
				Outputs: []types.IOObject{
					{ParamName: "result", DefineSID: FileMetadataSID},
				},
				Business: []types.DynamicConfigField{
					{
						ID:       "uploadRootDir",
						Name:     "Upload Root Directory",
						Type:     cnst.STRING,
						Desc:     "Directory where files will be saved",
						Required: false,
						Default:  "temp",
					},
				},
			},
		},
	})
}
