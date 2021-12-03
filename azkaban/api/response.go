package api

// TODO 不同版本的返回值
// TODO 返回值格式校验

type Project struct {
	ProjectId   int    `json:"projectId"`
	ProjectName string `json:"projectName"`
	CreatedBy   string `json:"createdBy"`
	//CreatedTime time.Time `json:"createdTime"`
	//userPermissions
	//groupPermissions
	Flows []Flow
}

type Flow struct {
	FlowId string `json:"flowId"`
}

type AuthenticateResponse struct {
	SessionId string `json:"session.id"`
	Status    string `json:"status"`
	Error     string `json:"error"`
}

type FetchUserProjectsResponse struct {
	Projects []Project `json:"projects"`
	Error    string    `json:"error"`
}

type FetchFlowsOfAProjectResponse struct {
	ProjectName string `json:"project"`
	ProjectId   int    `json:"projectId"`
	Flows       []Flow `json:"flows"`
	Error       string `json:"error"`
}

type ExecutionsResponse struct {
	ExecIds []int `json:"execIds"`
}

type ExecutionInformationResponse struct {
	Project    string         `json:"project"`
	UpdateTime int64          `json:"updateTime"`
	Attempt    int            `json:"attempt"`
	Execid     int            `json:"execid"`
	SubmitTime int64          `json:"submitTime"`
	Nodes      []OutsideNodes `json:"nodes"`
	NestedID   string         `json:"nestedId"`
	SubmitUser string         `json:"submitUser"`
	StartTime  int64          `json:"startTime"`
	ID         string         `json:"id"`
	EndTime    int64          `json:"endTime"`
	ProjectID  int            `json:"projectId"`
	FlowID     string         `json:"flowId"`
	Flow       string         `json:"flow"`
	Status     string         `json:"status"`
	//Type       interface{}    `json:"type"`
}

type OutsideNodes struct {
	Nodes      []InsideNodes `json:"nodes"`
	NestedID   string        `json:"nestedId"`
	StartTime  int64         `json:"startTime"`
	UpdateTime int64         `json:"updateTime"`
	ID         string        `json:"id"`
	EndTime    int64         `json:"endTime"`
	Type       string        `json:"type"`
	Attempt    int           `json:"attempt"`
	FlowID     string        `json:"flowId"`
	Flow       string        `json:"flow"`
	Status     string        `json:"status"`
}

type InsideNodes struct {
	NestedID   string   `json:"nestedId"`
	In         []string `json:"in,omitempty"`
	StartTime  int64    `json:"startTime"`
	UpdateTime int64    `json:"updateTime"`
	ID         string   `json:"id"`
	EndTime    int64    `json:"endTime"`
	Type       string   `json:"type"`
	Attempt    int      `json:"attempt"`
	Status     string   `json:"status"`
}
