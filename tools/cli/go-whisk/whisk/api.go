/*
 * Copyright 2015-2016 IBM Corporation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package whisk

import (
    "fmt"
    "net/http"
    "errors"
    "net/url"
    "../wski18n"
    "strings"
)

type ApiService struct {
    client *Client
}

type SendApi struct {
    ApiDoc         *Api      `json:"apidoc,omitempty"`
}

type Api struct {
    Namespace       string   `json:"namespace,omitempty"`
    ApiName         string   `json:"apiName,omitempty"`
    GatewayBasePath string   `json:"gatewayBasePath,omitempty"`
    GatewayRelPath  string   `json:"gatewayPath,omitempty"`
    GatewayMethod   string   `json:"gatewayMethod,omitempty"`
    Id              string   `json:"id,omitempty"`
    GatewayFullPath string   `json:"gatewayFullPath,omitempty"`
    Swagger         string   `json:"swagger,omitempty"`
    Action          *ApiAction `json:"action,omitempty"`
}

type ApiAction struct {
    Name            string    `json:"name,omitempty"`
    Namespace       string    `json:"namespace,omitempty"`
    BackendMethod   string    `json:"backendMethod,omitempty"`
    BackendUrl      string    `json:"backendUrl,omitempty"`
    Auth            string    `json:"authkey,omitempty"`
}

type ApiOptions struct {
    ActionName      string    `url:"action,omitempty"`
    ApiBasePath     string    `url:"basepath,omitempty"`
    ApiRelPath      string    `url:"relpath,omitempty"`
    ApiVerb         string    `url:"operation,omitempty"`
}

type ApiListOptions struct {
    ApiOptions
    Limit           int  `url:"limit"`
    Skip            int  `url:"skip"`
    Docs            bool `url:"docs,omitempty"`
}

type RetApi struct {
    Namespace       string    `json:"namespace"`
    Activated       bool      `json:"gwApiActivated"`
    TenantId        string    `json:"tenantId"`
    Swagger         *ApiSwagger `json:"apidoc,omitempty"`
}

type ApiSwagger struct {
    SwaggerName     string     `json:"swagger,omitempty"`
    BasePath        string     `json:"basePath,omitempty"`
    Info            *ApiSwaggerInfo `json:"info,omitempty"`
    Paths           map[string]map[string]map[string]map[string]interface{} `json:"paths,omitempty"`
}

type ApiSwaggerInfo struct {
    Title           string     `json:"title,omitempty"`
    Version         string     `json:"version,omitempty"`
}


var ApiVerbs map[string]bool = map[string]bool {
    "GET": true,
    "PUT": true,
    "POST": true,
    "DELETE": true,
}

////////////////////
// Api Methods //
////////////////////

func (s *ApiService) List(apiListOptions *ApiListOptions) ([]Api, *http.Response, error) {
    var route string
    var apiList []Api

    route = fmt.Sprintf("routes")

    routeUrl, err := addRouteOptions(route, apiListOptions)
    if err != nil {
        Debug(DbgError, "addRouteOptions(%s, %#v) error: '%s'\n", route, apiListOptions, err)
        errMsg := wski18n.T("Unable to add route options '{{.options}}'",
            map[string]interface{}{"options": apiListOptions})
        whiskErr := MakeWskErrorFromWskError(errors.New(errMsg), err, EXITCODE_ERR_GENERAL, DISPLAY_MSG,
            NO_DISPLAY_USAGE)
        return nil, nil, whiskErr
    }
    Debug(DbgError, "Api GET/list route with api options: %s\n", routeUrl)

    req, err := s.client.NewRequestUrl("GET", routeUrl, nil)
    if err != nil {
        Debug(DbgError, "http.NewRequest(GET, %s, nil) error: '%s'\n", routeUrl, err)
        errMsg := wski18n.T("Unable to create HTTP request for GET '{{.route}}': {{.err}}",
            map[string]interface{}{"route": routeUrl, "err": err})
        whiskErr := MakeWskErrorFromWskError(errors.New(errMsg), err, EXITCODE_ERR_NETWORK, DISPLAY_MSG,
            NO_DISPLAY_USAGE)
        return nil, nil, whiskErr
    }

    resp, err := s.client.Do(req, &apiList)
    if err != nil {
        Debug(DbgError, "s.client.Do() error - HTTP req %s; error '%s'\n", req.URL.String(), err)
        return nil, resp, err
    }

    return apiList, resp, err
}

func (s *ApiService) Insert(api *SendApi, overwrite bool) (*RetApi, *http.Response, error) {
    var sentAction interface{}

    route := fmt.Sprintf("routes")
    Debug(DbgInfo, "Api PUT route: %s\n", route)

    req, err := s.client.NewRequest("POST", route, api)
    if err != nil {
        Debug(DbgError, "http.NewRequest(POST, %s, %#v) error: '%s'\n", route, err, sentAction)
        errMsg := wski18n.T("Unable to create HTTP request for POST '{{.route}}': {{.err}}",
            map[string]interface{}{"route": route, "err": err})
        whiskErr := MakeWskErrorFromWskError(errors.New(errMsg), err, EXITCODE_ERR_NETWORK, DISPLAY_MSG,
            NO_DISPLAY_USAGE)
        return nil, nil, whiskErr
    }

    retApi := new(RetApi)
    resp, err := s.client.Do(req, &retApi)
    if err != nil {
        Debug(DbgError, "s.client.Do() error - HTTP req %s; error '%s'\n", req.URL.String(), err)
        return nil, resp, err
    }

    return retApi, resp, nil
}

func (s *ApiService) Get(api *Api, options *ApiOptions) (*RetApi, *http.Response, error) {
    // Encode resource name as a path (with no query ) before inserting it into the URI
    // This way any '?' chars in the name won't be treated as the beginning of the query params
    preEncodedApiId := api.Id
    encodedApiId := url.QueryEscape(preEncodedApiId) // Escape ':' and '/' characters typical in this id string
    apiId := (&url.URL{Path: encodedApiId}).String()
    route := fmt.Sprintf("routes/%s", apiId)
    Debug(DbgInfo, "Api GET route: %s\n", route)

    routeUrl, err := addRouteOptions(route, options)
    if err != nil {
        Debug(DbgError, "addRouteOptions(%s, %#v) error: '%s'\n", route, options, err)
        errMsg := wski18n.T("Unable to add route options '{{.options}}'",
            map[string]interface{}{"options": options})
        whiskErr := MakeWskErrorFromWskError(errors.New(errMsg), err, EXITCODE_ERR_GENERAL, DISPLAY_MSG,
            NO_DISPLAY_USAGE)
        return nil, nil, whiskErr
    }
    Debug(DbgError, "Api get route with options: %s\n", routeUrl)

    req, err := s.client.NewRequestUrl("GET", routeUrl, nil)
    if err != nil {
        Debug(DbgError, "http.NewRequest(GET, %s, nil) error: '%s'\n", route, err)
        errMsg := wski18n.T("Unable to create HTTP request for GET '{{.route}}': {{.err}}",
            map[string]interface{}{"route": route, "err": err})
        whiskErr := MakeWskErrorFromWskError(errors.New(errMsg), err, EXITCODE_ERR_NETWORK, DISPLAY_MSG,
            NO_DISPLAY_USAGE)
        return nil, nil, whiskErr
    }

    retApi := new(RetApi)
    resp, err := s.client.Do(req, &retApi)
    if err != nil {
        Debug(DbgError, "s.client.Do() error - HTTP req %s; error '%s'\n", req.URL.String(), err)
        return nil, resp, err
    }

    return retApi, resp, nil
}

func (s *ApiService) Delete(api *Api) (*http.Response, error) {
    // Encode resource name as a path (with no query ) before inserting it into the URI
    // This way any '?' chars in the name won't be treated as the beginning of the query params
    apiId := (&url.URL{Path: api.Id}).String()
    apiId = strings.Replace(apiId, "/", "!", -1)  // Since '/' is the URL path delimiter, replace these chars
    route := fmt.Sprintf("routes/%s", apiId)
    Debug(DbgInfo, "Api DELETE route: %s\n", route)

    req, err := s.client.NewRequest("DELETE", route, nil)
    if err != nil {
        Debug(DbgError, "http.NewRequest(DELETE, %s, nil) error: '%s'\n", route, err)
        errMsg := wski18n.T("Unable to create HTTP request for DELETE '{{.route}}': {{.err}}",
            map[string]interface{}{"route": route, "err": err})
        whiskErr := MakeWskErrorFromWskError(errors.New(errMsg), err, EXITCODE_ERR_NETWORK, DISPLAY_MSG,
            NO_DISPLAY_USAGE)
        return nil, whiskErr
    }

    retApi := new(Api)
    resp, err := s.client.Do(req, retApi)
    if err != nil {
        Debug(DbgError, "s.client.Do() error - HTTP req %s; error '%s'\n", req.URL.String(), err)
        return resp, err
    }

    return resp, nil
}

