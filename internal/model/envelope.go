package model

type Envelope struct {
	Version  string    `json:"version"`
	Mode     string    `json:"mode"`
	Command  string    `json:"command"`
	OK       bool      `json:"ok"`
	Proposal *Proposal `json:"proposal,omitempty"`
	Result   any       `json:"result"`
	Error    *Error    `json:"error"`
}

type Proposal struct {
	Action  string         `json:"action"`
	Target  map[string]any `json:"target,omitempty"`
	Inputs  map[string]any `json:"inputs,omitempty"`
	Effects []Effect       `json:"effects"`
}

type Effect struct {
	Type string `json:"type"`
	Kind string `json:"kind"`
	Path string `json:"path,omitempty"`
}

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
