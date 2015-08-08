package cmdproto

type MgoRequest struct {
	DB    string `json:"db"`
	DBCmd struct {
		DBCmd string   `json:"cmd"`
		Args  []string `json:"args,omitempty"`
	} `json:"cmd"`
	//more...
}

type MgoResponse struct {
}

//ScRequest
//used to start, stop, monitor(and other actions) a service.
type ScRequest struct {
	Op          string `json:"op"`
	ServiceInfo struct {
		Service string   `json:"service"`
		Args    []string `json:"args"`
	} `json:"si"`
}

//SysRequest
//used to execute nix command.
type SysRequest struct {
	Op   string   `json:"op"`
	Args []string `json:"args"`
}
