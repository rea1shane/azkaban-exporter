package api

import (
	"azkaban_exporter/util"
	"net/http"
	"strconv"
	"strings"
)

var singletonHttp = util.GetSingletonHttp()

// Authenticate return azkaban.Session's SessionId
// doc https://github.com/azkaban/azkaban/blob/master/docs/ajaxApi.rst#authenticate
func Authenticate(serverUrl string, username string, password string) (string, error) {
	method := "POST"
	response := Auth{}
	payload := strings.NewReader("action=login&username=" + username + "&password=" + password)
	req, err := http.NewRequest(method, serverUrl, payload)
	if err != nil {
		return "", err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	err = singletonHttp.Request(req, &response)
	if err != nil {
		return "", err
	}
	return response.SessionId, nil
}

// FetchUserProjects
// doc https://github.com/azkaban/azkaban/blob/master/docs/ajaxApi.rst#fetch-user-projects
func FetchUserProjects(serverUrl string, sessionId string) ([]Project, error) {
	method := "GET"
	response := UserProjects{}
	url := serverUrl + "/index?ajax=fetchuserprojects&session.id=" + sessionId
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}
	err = singletonHttp.Request(req, &response)
	if err != nil {
		return nil, err
	}
	return response.Projects, nil
}

// FetchFlowsOfAProject
// doc https://github.com/azkaban/azkaban/blob/master/docs/ajaxApi.rst#fetch-flows-of-a-project
func FetchFlowsOfAProject(serverUrl string, sessionId string, projectName string) ([]Flow, error) {
	method := "GET"
	response := ProjectFlows{}
	url := serverUrl + "/manager?ajax=fetchprojectflows&session.id=" + sessionId + "&project=" + projectName
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}
	err = singletonHttp.Request(req, &response)
	if err != nil {
		return nil, err
	}
	return response.Flows, nil
}

// FetchRunningExecutionsOfAFlow
// doc https://github.com/azkaban/azkaban/blob/master/docs/ajaxApi.rst#fetch-running-executions-of-a-flow
func FetchRunningExecutionsOfAFlow(serverUrl string, sessionId string, projectName string, flowId string) (Executions, error) {
	method := "GET"
	response := Executions{}
	url := serverUrl + "/executor?ajax=getRunning&session.id=" + sessionId + "&project=" + projectName + "&flow=" + flowId
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return Executions{}, err
	}
	err = singletonHttp.Request(req, &response)
	if err != nil {
		return Executions{}, err
	}
	return response, nil
}

// FetchAFlowExecution
// doc https://github.com/azkaban/azkaban/blob/master/docs/ajaxApi.rst#fetch-a-flow-execution
func FetchAFlowExecution(serverUrl string, sessionId string, execId int) (ExecutionInfo, error) {
	method := "GET"
	response := ExecutionInfo{}
	url := serverUrl + "/executor?ajax=fetchexecflow&session.id=" + sessionId + "&execid=" + strconv.Itoa(execId)
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return ExecutionInfo{}, err
	}
	err = singletonHttp.Request(req, &response)
	if err != nil {
		return ExecutionInfo{}, err
	}
	return response, nil
}
