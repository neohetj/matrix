package model

type RuntimeProfile string

const (
	ApplicationNodeType  = "ops/application"
	ServiceNodeType      = "ops/service"
	DatabaseNodeType     = "ops/database"
	MessageQueueNodeType = "ops/message_queue"
	NetworkNodeType      = "ops/network"
	VolumeNodeType       = "ops/volume"
	RunnerNodeType       = "ops/runner"
	MachineNodeType      = "ops/machine"
)

const (
	RuntimeProfileMatrixService       RuntimeProfile = "matrix-service"
	RuntimeProfileComposeService      RuntimeProfile = "compose-service"
	RuntimeProfileComposeDatabase     RuntimeProfile = "compose-database"
	RuntimeProfileComposeMessageQueue RuntimeProfile = "compose-message-queue"
	RuntimeProfileExternalManaged     RuntimeProfile = "external-managed"
)

type DeployAdapter string

const (
	DeployAdapterDockerCompose DeployAdapter = "docker-compose"
	DeployAdapterNone          DeployAdapter = "none"
)

// DeploymentSpec captures fields shared by deployable topology nodes.
type DeploymentSpec struct {
	RuntimeProfile RuntimeProfile `json:"runtimeProfile,omitempty"`
	DeployAdapter  DeployAdapter  `json:"deployAdapter,omitempty"`
	Deployable     bool           `json:"deployable,omitempty"`
	RunnerRef      string         `json:"runnerRef,omitempty"`
	ArtifactRef    string         `json:"artifactRef,omitempty"`
	Image          string         `json:"image,omitempty"`
	Ports          []string       `json:"ports,omitempty"`
	EnvRefs        []string       `json:"envRefs,omitempty"`
	SecretRefs     []string       `json:"secretRefs,omitempty"`
	NetworkRefs    []string       `json:"networkRefs,omitempty"`
	VolumeRefs     []string       `json:"volumeRefs,omitempty"`
	DependsOn      []string       `json:"dependsOn,omitempty"`
	EndpointRefs   []string       `json:"endpointRefs,omitempty"`
	RuleChainRefs  []string       `json:"ruleChainRefs,omitempty"`
}

type ApplicationConfig struct {
	Domain      string   `json:"domain,omitempty"`
	Environment string   `json:"environment,omitempty"`
	Owners      []string `json:"owners,omitempty"`
}

type ServiceConfig struct {
	Deployment  DeploymentSpec `json:"deployment,omitempty"`
	ServiceType string         `json:"serviceType,omitempty"`
	Language    string         `json:"language,omitempty"`
}

type DatabaseConfig struct {
	Deployment DeploymentSpec `json:"deployment,omitempty"`
	Engine     string         `json:"engine,omitempty"`
	Version    string         `json:"version,omitempty"`
}

type MessageQueueConfig struct {
	Deployment DeploymentSpec `json:"deployment,omitempty"`
	Engine     string         `json:"engine,omitempty"`
	Version    string         `json:"version,omitempty"`
	Queues     []string       `json:"queues,omitempty"`
	Topics     []string       `json:"topics,omitempty"`
}

type NetworkConfig struct {
	Driver string `json:"driver,omitempty"`
	Scope  string `json:"scope,omitempty"`
}

type VolumeConfig struct {
	Driver     string `json:"driver,omitempty"`
	Scope      string `json:"scope,omitempty"`
	MountPath  string `json:"mountPath,omitempty"`
	Persistent bool   `json:"persistent,omitempty"`
}

type RunnerConfig struct {
	ExecutorType string `json:"executorType,omitempty"`
	Environment  string `json:"environment,omitempty"`
	Address      string `json:"address,omitempty"`
}

type MachineConfig struct {
	Hostname        string   `json:"hostname,omitempty"`
	Address         string   `json:"address,omitempty"`
	OperatingSystem string   `json:"operatingSystem,omitempty"`
	Architecture    string   `json:"architecture,omitempty"`
	Labels          []string `json:"labels,omitempty"`
}
