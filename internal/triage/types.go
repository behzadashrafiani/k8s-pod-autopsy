package triage

type Signal string

const (
	SignalOOMKilled Signal = "OOMKilled"
	SignalCrashing  Signal = "Crashing"
	SignalWaiting   Signal = "Waiting" // non-starting: image pull / config errors
	SignalPending   Signal = "Pending" // unscheduled
	SignalSaturated Signal = "Saturated"
	SignalHealthy   Signal = "Healthy"
)

type ContainerRole string

const (
	RoleInit      ContainerRole = "init"
	RoleApp       ContainerRole = "app"
	RoleEphemeral ContainerRole = "ephemeral" // kubectl debug containers
)

// ContainerFinding is the culprit container's distilled state.
type ContainerFinding struct {
	Name         string
	Role         ContainerRole
	RestartCount int32
	// Last termination (empty for pure waiting states).
	ExitCode   int32
	ExitReason string // e.g. "OOMKilled", "Error"
	ExitPlain  string // plain-English exit code
	Message    string
	// Waiting state (empty for terminated states).
	WaitingReason  string
	WaitingMessage string
	HasMemLimit    bool
}

type OOMAnalysis struct {
	HitOwnLimit  bool
	NodePressure bool
	Verdict      string
}

type LogSection struct {
	Source      string   // "previous" or "current"
	Fallback    bool     // wanted previous, fell back to current
	Errors      []string // categorized error/panic lines
	Lines       []string // deduped tail
	Unavailable string   // reason if logs could not be fetched
}

type EventLine struct {
	Reason  string
	Message string
	Count   int32
}

type PodReport struct {
	Namespace  string
	Name       string
	Phase      string
	Signal     Signal
	Verdict    string
	Culprit    *ContainerFinding
	OOM        *OOMAnalysis
	Logs       *LogSection
	Saturation *SaturationReport
	Events     []EventLine
}

type ContainerSaturation struct {
	Name            string
	CPUUsed         string
	CPURequest      string
	CPULimit        string
	CPUPctLimit     float64 // -1 when no limit set
	MemUsed         string
	MemRequest      string
	MemLimit        string
	MemPctLimit     float64 // -1 when no limit set
	MemRisk         bool    // > 90% of mem limit
	CPUThrottleRisk bool    // > 90% of cpu limit (framed as "possible")
}

type SaturationReport struct {
	Namespace   string
	Name        string
	Containers  []ContainerSaturation
	Unavailable string
}

type CrashLoopReason string

const (
	ReasonCrashLoopBackOff CrashLoopReason = "CrashLoopBackOff"
	ReasonOOMKilled        CrashLoopReason = "OOMKilled"
	ReasonHighRestart      CrashLoopReason = "HighRestartCount"
)

type CrashLoopPod struct {
	Namespace    string
	Name         string
	Container    string
	Reason       CrashLoopReason
	RestartCount int32
}

type CrashLoopReport struct {
	Pods    []CrashLoopPod
	Scanned int
}
