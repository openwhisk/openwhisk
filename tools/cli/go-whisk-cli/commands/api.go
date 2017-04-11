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

package commands

import (
    "errors"
    "fmt"
    "reflect"
    "strconv"
    "strings"

    "../../go-whisk/whisk"
    "../wski18n"

    "github.com/fatih/color"
    "github.com/spf13/cobra"
    "encoding/json"
)

//////////////
// Commands //
//////////////

var apiExperimentalCmd = &cobra.Command{
    Use:   "api-experimental",
    Short: wski18n.T("work with APIs (experimental)"),
}

var apiCmd = &cobra.Command{
    Use:   "api",
    Short: wski18n.T("COMING SOON - work with APIs"),
}

var apiCreateCmd = &cobra.Command{
    Use:           "create ([BASE_PATH] API_PATH API_VERB ACTION] | --config-file CFG_FILE) ",
    Short:         wski18n.T("create a new API"),
    SilenceUsage:  true,
    SilenceErrors: true,
    PreRunE:       setupClientConfig,
    RunE: func(cmd *cobra.Command, args []string) error {

        var api *whisk.Api
        var err error

        if (len(args) == 0 && flags.api.configfile == "") {
            whisk.Debug(whisk.DbgError, "No swagger file and no arguments\n")
            errMsg := wski18n.T("Invalid argument(s). Specify a swagger file or specify an API base path with an API path, an API verb, and an action name.")
            whiskErr := whisk.MakeWskError(errors.New(errMsg), whisk.EXITCODE_ERR_GENERAL,
                whisk.DISPLAY_MSG, whisk.DISPLAY_USAGE)
            return whiskErr
        } else if (len(args) == 0 && flags.api.configfile != "") {
            api, err = parseSwaggerApi()
            if err != nil {
                whisk.Debug(whisk.DbgError, "parseSwaggerApi() error: %s\n", err)
                errMsg := wski18n.T("Unable to parse swagger file: {{.err}}", map[string]interface{}{"err": err})
                whiskErr := whisk.MakeWskErrorFromWskError(errors.New(errMsg), err, whisk.EXITCODE_ERR_GENERAL,
                    whisk.DISPLAY_MSG, whisk.DISPLAY_USAGE)
                return whiskErr
            }
        } else {
            if whiskErr := checkArgs(args, 3, 4, "Api create",
                wski18n.T("Specify a swagger file or specify an API base path with an API path, an API verb, and an action name.")); whiskErr != nil {
                return whiskErr
            }
            api, err = parseApi(cmd, args)
            if err != nil {
                whisk.Debug(whisk.DbgError, "parseApi(%s, %s) error: %s\n", cmd, args, err)
                errMsg := wski18n.T("Unable to parse api command arguments: {{.err}}",
                    map[string]interface{}{"err": err})
                whiskErr := whisk.MakeWskErrorFromWskError(errors.New(errMsg), err, whisk.EXITCODE_ERR_GENERAL,
                    whisk.DISPLAY_MSG, whisk.DISPLAY_USAGE)
                return whiskErr
            }
        }

        apiCreateReq := new(whisk.ApiCreateRequest)
        apiCreateReq.ApiDoc = api
        apiCreateReqOptions := new(whisk.ApiCreateRequestOptions)
        retApi, _, err := client.Apis.Insert(apiCreateReq, apiCreateReqOptions, whisk.DoNotOverwrite)
        if err != nil {
            whisk.Debug(whisk.DbgError, "client.Apis.Insert(%#v, false) error: %s\n", api, err)
            errMsg := wski18n.T("Unable to create API: {{.err}}", map[string]interface{}{"err": err})
            whiskErr := whisk.MakeWskErrorFromWskError(errors.New(errMsg), err, whisk.EXITCODE_ERR_NETWORK,
                whisk.DISPLAY_MSG, whisk.NO_DISPLAY_USAGE)
            return whiskErr
        }

        if (api.Swagger == "") {
            baseUrl := retApi.BaseUrl
            fmt.Fprintf(color.Output,
                wski18n.T("{{.ok}} created API {{.path}} {{.verb}} for action {{.name}}\n{{.fullpath}}\n",
                    map[string]interface{}{
                        "ok": color.GreenString("ok:"),
                        "path": strings.TrimSuffix(api.GatewayBasePath, "/")+api.GatewayRelPath,
                        "verb": api.GatewayMethod,
                        "name": boldString("/"+api.Action.Namespace+"/"+api.Action.Name),
                        "fullpath": strings.TrimSuffix(baseUrl, "/")+api.GatewayRelPath,
                    }))
        } else {
            whisk.Debug(whisk.DbgInfo, "Processing swagger based create API response\n")
            baseUrl := retApi.BaseUrl
            for path, _ := range retApi.Swagger.Paths {
                managedUrl := strings.TrimSuffix(baseUrl, "/")+path
                whisk.Debug(whisk.DbgInfo, "Managed path: %s\n",managedUrl)
                for op, opv  := range retApi.Swagger.Paths[path] {
                    whisk.Debug(whisk.DbgInfo, "Path operation: %s\n", op)
                    fmt.Fprintf(color.Output,
                        wski18n.T("{{.ok}} created API {{.path}} {{.verb}} for action {{.name}}\n{{.fullpath}}\n",
                            map[string]interface{}{
                                "ok": color.GreenString("ok:"),
                                "path": path,
                                "verb": op,
                                "name": boldString(opv.XOpenWhisk.ActionName),
                                "fullpath": managedUrl,
                            }))
                }
            }
        }


        return nil
    },
}

//var apiUpdateCmd = &cobra.Command{
//    Use:           "update API_PATH API_VERB ACTION",
//    Short:         wski18n.T("update an existing API"),
//    SilenceUsage:  true,
//    SilenceErrors: true,
//    PreRunE:       setupClientConfig,
//    RunE: func(cmd *cobra.Command, args []string) error {
//
//        if whiskErr := checkArgs(args, 3, 3, "Api update",
//            wski18n.T("An API path, an API verb, and an action name are required.")); whiskErr != nil {
//            return whiskErr
//        }
//
//        api, err := parseApi(cmd, args)
//        if err != nil {
//            whisk.Debug(whisk.DbgError, "parseApi(%s, %s) error: %s\n", cmd, args, err)
//            errMsg := wski18n.T("Unable to parse API command arguments: {{.err}}", map[string]interface{}{"err": err})
//            whiskErr := whisk.MakeWskErrorFromWskError(errors.New(errMsg), err, whisk.EXITCODE_ERR_GENERAL,
//                whisk.DISPLAY_MSG, whisk.DISPLAY_USAGE)
//            return whiskErr
//        }
//        sendApi := new(whisk.ApiCreateRequest)
//        sendApi.ApiDoc = api
//
//        retApi, _, err := client.Apis.Insert(sendApi, true)
//        if err != nil {
//            whisk.Debug(whisk.DbgError, "client.Apis.Insert(%#v, %t, false) error: %s\n", api, err)
//            errMsg := wski18n.T("Unable to update API: {{.err}}", map[string]interface{}{"err": err})
//            whiskErr := whisk.MakeWskErrorFromWskError(errors.New(errMsg), err, whisk.EXITCODE_ERR_NETWORK,
//                whisk.DISPLAY_MSG, whisk.NO_DISPLAY_USAGE)
//            return whiskErr
//        }
//
//        fmt.Fprintf(color.Output,
//            wski18n.T("{{.ok}} updated API {{.path}} {{.verb}} for action {{.name}}\n{{.fullpath}}\n",
//                map[string]interface{}{
//                    "ok": color.GreenString("ok:"),
//                    "path": api.GatewayRelPath,
//                    "verb": api.GatewayMethod,
//                    "name": boldString("/"+api.Action.Name),
//                    "fullpath": getManagedUrl(retApi, api.GatewayRelPath, api.GatewayMethod),
//                }))
//        return nil
//    },
//}

var apiGetCmd = &cobra.Command{
    Use:           "get BASE_PATH | API_NAME",
    Short:         wski18n.T("get API details"),
    SilenceUsage:  true,
    SilenceErrors: true,
    PreRunE:       setupClientConfig,
    RunE: func(cmd *cobra.Command, args []string) error {
        var err error
        var isBasePathArg bool = true

        if whiskErr := checkArgs(args, 1, 1, "Api get",
            wski18n.T("An API base path or API name is required.")); whiskErr != nil {
            return whiskErr
        }

        apiGetReq := new(whisk.ApiGetRequest)
        apiGetReqOptions := new(whisk.ApiGetRequestOptions)
        apiGetReqOptions.ApiBasePath = args[0]
        retApi, _, err := client.Apis.Get(apiGetReq, apiGetReqOptions)
        if err != nil {
            whisk.Debug(whisk.DbgError, "client.Apis.Get(%#v, %#v) error: %s\n", apiGetReq, apiGetReqOptions, err)
            errMsg := wski18n.T("Unable to get API '{{.name}}': {{.err}}", map[string]interface{}{"name": args[0], "err": err})
            whiskErr := whisk.MakeWskErrorFromWskError(errors.New(errMsg), err, whisk.EXITCODE_ERR_GENERAL,
                whisk.DISPLAY_MSG, whisk.NO_DISPLAY_USAGE)
            return whiskErr
        }
        whisk.Debug(whisk.DbgInfo, "client.Apis.Get returned: %#v\n", retApi)

        var displayResult interface{} = nil
        if (flags.common.detail) {
            if (retApi.Apis != nil && len(retApi.Apis) > 0 &&
                retApi.Apis[0].ApiValue != nil) {
                displayResult = retApi.Apis[0].ApiValue
            } else {
                whisk.Debug(whisk.DbgError, "No result object returned\n")
            }
        } else {
            if (retApi.Apis != nil && len(retApi.Apis) > 0 &&
                retApi.Apis[0].ApiValue != nil &&
                retApi.Apis[0].ApiValue.Swagger != nil) {
                  displayResult = retApi.Apis[0].ApiValue.Swagger
            } else {
                  whisk.Debug(whisk.DbgError, "No swagger returned\n")
            }
        }
        if (displayResult == nil) {
            var errMsg string
            if (isBasePathArg) {
                errMsg = wski18n.T("API does not exist for basepath {{.basepath}}",
                    map[string]interface{}{"basepath": args[0]})
            } else {
                errMsg = wski18n.T("API does not exist for API name {{.apiname}}",
                    map[string]interface{}{"apiname": args[0]})
            }

            whiskErr := whisk.MakeWskError(errors.New(errMsg), whisk.EXITCODE_ERR_GENERAL,
                whisk.DISPLAY_MSG, whisk.NO_DISPLAY_USAGE)
            return whiskErr
        }
        printJSON(displayResult)

        return nil
    },
}

var apiDeleteCmd = &cobra.Command{
    Use:           "delete BASE_PATH | API_NAME [API_PATH [API_VERB]]",
    Short:         wski18n.T("delete an API"),
    SilenceUsage:  true,
    SilenceErrors: true,
    PreRunE:       setupClientConfig,
    RunE:          func(cmd *cobra.Command, args []string) error {
        if whiskErr := checkArgs(args, 1, 3, "Api delete",
            wski18n.T("An API base path or API name is required.  An optional API relative path and operation may also be provided.")); whiskErr != nil {
            return whiskErr
        }

        apiDeleteReq := new(whisk.ApiDeleteRequest)
        apiDeleteReqOptions := new(whisk.ApiDeleteRequestOptions)
        // Is the argument a basepath (must start with /) or an API name
        if _, ok := isValidBasepath(args[0]); !ok {
            whisk.Debug(whisk.DbgInfo, "Treating '%s' as an API name; as it does not begin with '/'\n", args[0])
            apiDeleteReqOptions.ApiBasePath = args[0]
        } else {
            apiDeleteReqOptions.ApiBasePath = args[0]
        }

        if (len(args) > 1) {
            // Is the API path valid?
            if whiskErr, ok := isValidRelpath(args[1]); !ok {
                return whiskErr
            }
            apiDeleteReqOptions.ApiRelPath = args[1]
        }
        if (len(args) > 2) {
            // Is the API verb valid?
            if whiskErr, ok := IsValidApiVerb(args[2]); !ok {
                return whiskErr
            }
            apiDeleteReqOptions.ApiVerb = strings.ToUpper(args[2])
        }

        _, err := client.Apis.Delete(apiDeleteReq, apiDeleteReqOptions)
        if err != nil {
            whisk.Debug(whisk.DbgError, "client.Apis.Delete(%#v, %#v) error: %s\n", apiDeleteReq, apiDeleteReqOptions, err)
            errMsg := wski18n.T("Unable to delete API: {{.err}}", map[string]interface{}{"err": err})
            whiskErr := whisk.MakeWskErrorFromWskError(errors.New(errMsg), err, whisk.EXITCODE_ERR_GENERAL,
                whisk.DISPLAY_MSG, whisk.NO_DISPLAY_USAGE)
            return whiskErr
        }

        if (len(args) == 1) {
            fmt.Fprintf(color.Output,
                wski18n.T("{{.ok}} deleted API {{.basepath}}\n",
                    map[string]interface{}{
                        "ok": color.GreenString("ok:"),
                        "basepath": apiDeleteReqOptions.ApiBasePath,
                    }))
        } else if (len(args) == 2 ) {
            fmt.Fprintf(color.Output,
                wski18n.T("{{.ok}} deleted {{.path}} from {{.basepath}}\n",
                    map[string]interface{}{
                        "ok": color.GreenString("ok:"),
                        "path": apiDeleteReqOptions.ApiRelPath,
                        "basepath": apiDeleteReqOptions.ApiBasePath,
                    }))
        } else {
            fmt.Fprintf(color.Output,
                wski18n.T("{{.ok}} deleted {{.path}} {{.verb}} from {{.basepath}}\n",
                    map[string]interface{}{
                        "ok": color.GreenString("ok:"),
                        "path": apiDeleteReqOptions.ApiRelPath,
                        "verb": apiDeleteReqOptions.ApiVerb,
                        "basepath": apiDeleteReqOptions.ApiBasePath,
                    }))
        }

        return nil
    },
}

var fmtString = "%-30s %7s %20s  %s\n"
var apiListCmd = &cobra.Command{
    Use:           "list [[BASE_PATH | API_NAME] [API_PATH [API_VERB]]",
    Short:         wski18n.T("list APIs"),
    SilenceUsage:  true,
    SilenceErrors: true,
    PreRunE:       setupClientConfig,
    RunE: func(cmd *cobra.Command, args []string) error {
        var err error
        var retApiList *whisk.ApiListResponse
        var retApi *whisk.ApiGetResponse
        var retApiArray *whisk.RetApiArray

        if whiskErr := checkArgs(args, 0, 3, "Api list",
            wski18n.T("Optional parameters are: API base path (or API name), API relative path and operation.")); whiskErr != nil {
            return whiskErr
        }

        // Get API request body
        apiGetReq := new(whisk.ApiGetRequest)
        apiGetReq.Namespace = client.Config.Namespace

        // Get API request options
        apiGetReqOptions := new(whisk.ApiGetRequestOptions)

        // List API request query parameters
        apiListReqOptions := new(whisk.ApiListRequestOptions)
        apiListReqOptions.Limit = flags.common.limit
        apiListReqOptions.Skip = flags.common.skip

        if (len(args) == 0) {
            retApiList, _, err = client.Apis.List(apiListReqOptions)
            if err != nil {
                whisk.Debug(whisk.DbgError, "client.Apis.List(%#v) error: %s\n", apiListReqOptions, err)
                errMsg := wski18n.T("Unable to obtain the API list: {{.err}}", map[string]interface{}{"err": err})
                whiskErr := whisk.MakeWskErrorFromWskError(errors.New(errMsg), err, whisk.EXITCODE_ERR_GENERAL,
                    whisk.DISPLAY_MSG, whisk.NO_DISPLAY_USAGE)
                return whiskErr
            }
            whisk.Debug(whisk.DbgInfo, "client.Apis.List returned: %#v (%+v)\n", retApiList, retApiList)
            // Cast to a common type to allow for code to print out apilist response or apiget response
            retApiArray = (*whisk.RetApiArray)(retApiList)
        } else {
            // The first argument is either a basepath (must start with /) or an API name
            apiGetReqOptions.ApiBasePath = args[0]
            if (len(args) > 1) {
                // Is the API path valid?
                if whiskErr, ok := isValidRelpath(args[1]); !ok {
                    return whiskErr
                }
                apiGetReqOptions.ApiRelPath = args[1]
            }
            if (len(args) > 2) {
                // Is the API verb valid?
                if whiskErr, ok := IsValidApiVerb(args[2]); !ok {
                    return whiskErr
                }
                apiGetReqOptions.ApiVerb = strings.ToUpper(args[2])
            }

            retApi, _, err = client.Apis.Get(apiGetReq, apiGetReqOptions)
            if err != nil {
                whisk.Debug(whisk.DbgError, "client.Apis.Get(%#v, %#v) error: %s\n", apiGetReq, apiGetReqOptions, err)
                errMsg := wski18n.T("Unable to obtain the API list: {{.err}}", map[string]interface{}{"err": err})
                whiskErr := whisk.MakeWskErrorFromWskError(errors.New(errMsg), err, whisk.EXITCODE_ERR_GENERAL,
                    whisk.DISPLAY_MSG, whisk.NO_DISPLAY_USAGE)
                return whiskErr
            }
            whisk.Debug(whisk.DbgInfo, "client.Apis.Get returned: %#v\n", retApi)
            // Cast to a common type to allow for code to print out apilist response or apiget response
            retApiArray = (*whisk.RetApiArray)(retApi)
        }

        // Display the APIs - applying any specified filtering
        if (flags.common.full) {
            fmt.Fprintf(color.Output,
                wski18n.T("{{.ok}} APIs\n",
                    map[string]interface{}{
                        "ok": color.GreenString("ok:"),
                    }))

            for i:=0; i<len(retApiArray.Apis); i++ {
                printFilteredListApi(retApiArray.Apis[i].ApiValue, (*whisk.ApiOptions)(apiGetReqOptions))
            }
        } else {
            // Dynamically create the output format string based on the maximum size of the
            // fully qualified action name and the API Name.
            maxActionNameSize := min(40, max(len("Action"), getLargestActionNameSize(retApiArray, (*whisk.ApiOptions)(apiGetReqOptions))))
            maxApiNameSize := min(30, max(len("API Name"), getLargestApiNameSize(retApiArray, (*whisk.ApiOptions)(apiGetReqOptions))))
            fmtString = "%-"+strconv.Itoa(maxActionNameSize)+"s %7s %"+strconv.Itoa(maxApiNameSize+1)+"s  %s\n"

            fmt.Fprintf(color.Output,
                wski18n.T("{{.ok}} APIs\n",
                    map[string]interface{}{
                        "ok": color.GreenString("ok:"),
                    }))
            fmt.Printf(fmtString, "Action", "Verb", "API Name", "URL")

            for i:=0; i<len(retApiArray.Apis); i++ {
                printFilteredListRow(retApiArray.Apis[i].ApiValue, (*whisk.ApiOptions)(apiGetReqOptions), maxActionNameSize, maxApiNameSize)
            }
        }

        return nil
    },
}

/*
 * Takes an API object (containing one more more single basepath/relpath/operation triplets)
 * and some filtering configuration.  For each API endpoint matching the filtering criteria, display
 * each endpoint's configuration - one line per configuration property (action name, verb, api name, api gw url)
 */
func printFilteredListApi(resultApi *whisk.RetApi, api *whisk.ApiOptions) {
    baseUrl := strings.TrimSuffix(resultApi.BaseUrl, "/")
    apiName := resultApi.Swagger.Info.Title
    basePath := resultApi.Swagger.BasePath
    if (resultApi.Swagger != nil && resultApi.Swagger.Paths != nil) {
        for path, _ := range resultApi.Swagger.Paths {
            whisk.Debug(whisk.DbgInfo, "apiGetCmd: comparing api relpath: %s\n", path)
            if ( len(api.ApiRelPath) == 0 || path == api.ApiRelPath) {
                whisk.Debug(whisk.DbgInfo, "apiGetCmd: relpath matches\n")
                for op, opv  := range resultApi.Swagger.Paths[path] {
                    whisk.Debug(whisk.DbgInfo, "apiGetCmd: comparing operation: '%s'\n", op)
                    if ( len(api.ApiVerb) == 0 || strings.ToLower(op) == strings.ToLower(api.ApiVerb)) {
                        whisk.Debug(whisk.DbgInfo, "apiGetCmd: operation matches: %#v\n", opv)
                        var actionName = "/"+opv.XOpenWhisk.Namespace+"/"+opv.XOpenWhisk.ActionName
                        fmt.Printf("%s: %s\n", wski18n.T("Action"), actionName)
                        fmt.Printf("  %s: %s\n", wski18n.T("API Name"), apiName)
                        fmt.Printf("  %s: %s\n", wski18n.T("Base path"), basePath)
                        fmt.Printf("  %s: %s\n", wski18n.T("Path"), path)
                        fmt.Printf("  %s: %s\n", wski18n.T("Verb"), op)
                        fmt.Printf("  %s: %s\n", wski18n.T("URL"), baseUrl+path)
                    }
                }
            }
        }
    }
}

/*
 * Takes an API object (containing one more more single basepath/relpath/operation triplets)
 * and some filtering configuration.  For each API matching the filtering criteria, display the API
 * on a single line (action name, verb, api name, api gw url).
 *
 * NOTE: Large action name and api name value will be truncated by their associated max size parameters.
 */
func printFilteredListRow(resultApi *whisk.RetApi, api *whisk.ApiOptions, maxActionNameSize int, maxApiNameSize int) {
    baseUrl := strings.TrimSuffix(resultApi.BaseUrl, "/")
    apiName := resultApi.Swagger.Info.Title
    if (resultApi.Swagger != nil && resultApi.Swagger.Paths != nil) {
        for path, _ := range resultApi.Swagger.Paths {
            whisk.Debug(whisk.DbgInfo, "apiGetCmd: comparing api relpath: %s\n", path)
            if ( len(api.ApiRelPath) == 0 || path == api.ApiRelPath) {
                whisk.Debug(whisk.DbgInfo, "apiGetCmd: relpath matches\n")
                for op, opv  := range resultApi.Swagger.Paths[path] {
                    whisk.Debug(whisk.DbgInfo, "apiGetCmd: comparing operation: '%s'\n", op)
                    if ( len(api.ApiVerb) == 0 || strings.ToLower(op) == strings.ToLower(api.ApiVerb)) {
                        whisk.Debug(whisk.DbgInfo, "apiGetCmd: operation matches: %#v\n", opv)
                        var actionName = "/"+opv.XOpenWhisk.Namespace+"/"+opv.XOpenWhisk.ActionName
                        fmt.Printf(fmtString,
                            actionName[0 : min(len(actionName), maxActionNameSize)],
                            op,
                            apiName[0 : min(len(apiName), maxApiNameSize)],
                            baseUrl+path)
                    }
                }
            }
        }
    }
}

func getLargestActionNameSize(retApiArray *whisk.RetApiArray, api *whisk.ApiOptions) int {
    var maxNameSize = 0
    for i:=0; i<len(retApiArray.Apis); i++ {
        var resultApi = retApiArray.Apis[i].ApiValue
        if (resultApi.Swagger != nil && resultApi.Swagger.Paths != nil) {
            for path, _ := range resultApi.Swagger.Paths {
                whisk.Debug(whisk.DbgInfo, "getLargestActionNameSize: comparing api relpath: %s\n", path)
                if ( len(api.ApiRelPath) == 0 || path == api.ApiRelPath) {
                    whisk.Debug(whisk.DbgInfo, "getLargestActionNameSize: relpath matches\n")
                    for op, opv  := range resultApi.Swagger.Paths[path] {
                        whisk.Debug(whisk.DbgInfo, "getLargestActionNameSize: comparing operation: '%s'\n", op)
                        if ( len(api.ApiVerb) == 0 || strings.ToLower(op) == strings.ToLower(api.ApiVerb)) {
                            whisk.Debug(whisk.DbgInfo, "getLargestActionNameSize: operation matches: %#v\n", opv)
                            var fullActionName = "/"+opv.XOpenWhisk.Namespace+"/"+opv.XOpenWhisk.ActionName
                            if (len(fullActionName) > maxNameSize) {
                                maxNameSize = len(fullActionName)
                            }
                        }
                    }
                }
            }
        }
    }
    return maxNameSize
}

func getLargestApiNameSize(retApiArray *whisk.RetApiArray, api *whisk.ApiOptions) int {
    var maxNameSize = 0
    for i:=0; i<len(retApiArray.Apis); i++ {
        var resultApi = retApiArray.Apis[i].ApiValue
        apiName := resultApi.Swagger.Info.Title
        if (resultApi.Swagger != nil && resultApi.Swagger.Paths != nil) {
            for path, _ := range resultApi.Swagger.Paths {
                whisk.Debug(whisk.DbgInfo, "getLargestActionNameSize: comparing api relpath: %s\n", path)
                if ( len(api.ApiRelPath) == 0 || path == api.ApiRelPath) {
                    whisk.Debug(whisk.DbgInfo, "getLargestActionNameSize: relpath matches\n")
                    for op, opv  := range resultApi.Swagger.Paths[path] {
                        whisk.Debug(whisk.DbgInfo, "getLargestActionNameSize: comparing operation: '%s'\n", op)
                        if ( len(api.ApiVerb) == 0 || strings.ToLower(op) == strings.ToLower(api.ApiVerb)) {
                            whisk.Debug(whisk.DbgInfo, "getLargestActionNameSize: operation matches: %#v\n", opv)
                            if (len(apiName) > maxNameSize) {
                                maxNameSize = len(apiName)
                            }
                        }
                    }
                }
            }
        }
    }
    return maxNameSize
}

/*
 * if # args = 4
 * args[0] = API base path
 * args[0] = API relative path
 * args[1] = API verb
 * args[2] = Optional.  Action name (may or may not be qualified with namespace and package name)
 *
 * if # args = 3
 * args[0] = API relative path
 * args[1] = API verb
 * args[2] = Optional.  Action name (may or may not be qualified with namespace and package name)
 */
func parseApi(cmd *cobra.Command, args []string) (*whisk.Api, error) {
    var err error
    var basepath string = "/"
    var apiname string
    var basepathArgIsApiName = false;

    api := new(whisk.Api)

    if (len(args) > 3) {
        // Is the argument a basepath (must start with /) or an API name
        if _, ok := isValidBasepath(args[0]); !ok {
            whisk.Debug(whisk.DbgInfo, "Treating '%s' as an API name; as it does not begin with '/'\n", args[0])
            basepathArgIsApiName = true;
        }
        basepath = args[0]

        // Shift the args so the remaining code works with or without the explicit base path arg
        args = args[1:]
    }

    // Is the API path valid?
    if (len(args) > 0) {
        if whiskErr, ok := isValidRelpath(args[0]); !ok {
            return nil, whiskErr
        }
        api.GatewayRelPath = args[0]    // Maintain case as URLs may be case-sensitive
    }

    // Is the API verb valid?
    if (len(args) > 1) {
        if whiskErr, ok := IsValidApiVerb(args[1]); !ok {
            return nil, whiskErr
        }
        api.GatewayMethod = strings.ToUpper(args[1])
    }

    // Is the specified action name valid?
    var qName QualifiedName
    if (len(args) == 3) {
        qName = QualifiedName{}
        qName, err = parseQualifiedName(args[2])
        if err != nil {
            whisk.Debug(whisk.DbgError, "parseQualifiedName(%s) failed: %s\n", args[2], err)
            errMsg := wski18n.T("'{{.name}}' is not a valid action name: {{.err}}",
                map[string]interface{}{"name": args[2], "err": err})
            whiskErr := whisk.MakeWskErrorFromWskError(errors.New(errMsg), err, whisk.EXITCODE_ERR_GENERAL,
                whisk.DISPLAY_MSG, whisk.DISPLAY_USAGE)
            return nil, whiskErr
        }
        if (qName.entityName == "") {
            whisk.Debug(whisk.DbgError, "Action name '%s' is invalid\n", args[2])
            errMsg := wski18n.T("'{{.name}}' is not a valid action name.", map[string]interface{}{"name": args[2]})
            whiskErr := whisk.MakeWskErrorFromWskError(errors.New(errMsg), err, whisk.EXITCODE_ERR_GENERAL,
                whisk.DISPLAY_MSG, whisk.DISPLAY_USAGE)
            return nil, whiskErr
        }
    }

    if ( len(flags.api.apiname) > 0 ) {
        if (basepathArgIsApiName) {
            // Specifying API name as argument AND as a --apiname option value is invalid
            whisk.Debug(whisk.DbgError, "API is specified as an argument '%s' and as a flag '%s'\n", basepath, flags.api.apiname)
            errMsg := wski18n.T("An API name can only be specified once.")
            whiskErr := whisk.MakeWskError(errors.New(errMsg), whisk.EXITCODE_ERR_GENERAL,
                whisk.DISPLAY_MSG, whisk.DISPLAY_USAGE)
            return nil, whiskErr
        }
        apiname = flags.api.apiname
    }

    api.Namespace = client.Config.Namespace
    api.Action = new(whisk.ApiAction)
    api.Action.BackendUrl = "https://" + client.Config.Host + "/api/v1/namespaces/" + qName.namespace + "/actions/" + qName.entityName
    api.Action.BackendMethod = "POST"
    api.Action.Name = qName.entityName
    api.Action.Namespace = qName.namespace
    api.Action.Auth = client.Config.AuthToken
    api.ApiName = apiname
    api.GatewayBasePath = basepath
    if (!basepathArgIsApiName) { api.Id = "API:"+api.Namespace+":"+api.GatewayBasePath }

    whisk.Debug(whisk.DbgInfo, "Parsed api struct: %#v\n", api)
    return api, nil
}

func parseSwaggerApi() (*whisk.Api, error) {
    // Test is for completeness, but this situation should only arise due to an internal error
    if ( len(flags.api.configfile) == 0 ) {
        whisk.Debug(whisk.DbgError, "No swagger file is specified\n")
        errMsg := wski18n.T("A configuration file was not specified.")
        whiskErr := whisk.MakeWskError(errors.New(errMsg),whisk.EXITCODE_ERR_GENERAL,
            whisk.DISPLAY_MSG, whisk.DISPLAY_USAGE)
        return nil, whiskErr
    }

    swagger, err:= readFile(flags.api.configfile)
    if ( err != nil ) {
        whisk.Debug(whisk.DbgError, "readFile(%s) error: %s\n", flags.api.configfile, err)
        errMsg := wski18n.T("Error reading swagger file '{{.name}}': {{.err}}",
                map[string]interface{}{"name": flags.api.configfile, "err": err})
        whiskErr := whisk.MakeWskErrorFromWskError(errors.New(errMsg), err, whisk.EXITCODE_ERR_GENERAL,
            whisk.DISPLAY_MSG, whisk.DISPLAY_USAGE)
        return nil, whiskErr
    }

    // Parse the JSON into a swagger object
    swaggerObj := new(whisk.ApiSwagger)
    err = json.Unmarshal([]byte(swagger), swaggerObj)
    if ( err != nil ) {
        whisk.Debug(whisk.DbgError, "JSON parse of `%s' error: %s\n", flags.api.configfile, err)
        errMsg := wski18n.T("Error parsing swagger file '{{.name}}': {{.err}}",
                map[string]interface{}{"name": flags.api.configfile, "err": err})
        whiskErr := whisk.MakeWskErrorFromWskError(errors.New(errMsg), err, whisk.EXITCODE_ERR_GENERAL,
            whisk.DISPLAY_MSG, whisk.DISPLAY_USAGE)
        return nil, whiskErr
    }
    if (swaggerObj.BasePath == "" || swaggerObj.SwaggerName == "" || swaggerObj.Info == nil || swaggerObj.Paths == nil) {
        whisk.Debug(whisk.DbgError, "Swagger file is invalid.\n", flags.api.configfile, err)
        errMsg := wski18n.T("Swagger file is invalid (missing basePath, info, paths, or swagger fields)")
        whiskErr := whisk.MakeWskError(errors.New(errMsg), whisk.EXITCODE_ERR_GENERAL,
            whisk.DISPLAY_MSG, whisk.DISPLAY_USAGE)
        return nil, whiskErr
    }
    if _, ok := isValidBasepath(swaggerObj.BasePath); !ok {
        whisk.Debug(whisk.DbgError, "Swagger file basePath is invalid.\n", flags.api.configfile, err)
        errMsg := wski18n.T("Swagger file basePath must start with a leading slash (/)")
        whiskErr := whisk.MakeWskError(errors.New(errMsg), whisk.EXITCODE_ERR_GENERAL,
            whisk.DISPLAY_MSG, whisk.DISPLAY_USAGE)
        return nil, whiskErr
    }

    api := new(whisk.Api)
    api.Namespace = client.Config.Namespace
    api.Swagger = swagger

    return api, nil
}

func IsValidApiVerb(verb string) (error, bool) {
    // Is the API verb valid?
    if _, ok := whisk.ApiVerbs[strings.ToUpper(verb)]; !ok {
        whisk.Debug(whisk.DbgError, "Invalid API verb: %s\n", verb)
        errMsg := wski18n.T("'{{.verb}}' is not a valid API verb.  Valid values are: {{.verbs}}",
                map[string]interface{}{
                    "verb": verb,
                    "verbs": reflect.ValueOf(whisk.ApiVerbs).MapKeys()})
        whiskErr := whisk.MakeWskError(errors.New(errMsg), whisk.EXITCODE_ERR_GENERAL,
            whisk.DISPLAY_MSG, whisk.DISPLAY_USAGE)
        return whiskErr, false
    }
    return nil, true
}

func hasPathPrefix(path string) (error, bool) {
    if (! strings.HasPrefix(path, "/")) {
        whisk.Debug(whisk.DbgError, "path does not begin with '/': %s\n", path)
        errMsg := wski18n.T("'{{.path}}' must begin with '/'.",
                map[string]interface{}{
                    "path": path,
                })
        whiskErr := whisk.MakeWskError(errors.New(errMsg), whisk.EXITCODE_ERR_GENERAL,
            whisk.DISPLAY_MSG, whisk.DISPLAY_USAGE)
        return whiskErr, false
    }
    return nil, true
}

func isValidBasepath(basepath string) (error, bool) {
    if whiskerr, ok := hasPathPrefix(basepath); !ok {
        return whiskerr, false
    }
    return nil, true
}

func isValidRelpath(relpath string) (error, bool) {
    if whiskerr, ok := hasPathPrefix(relpath); !ok {
        return whiskerr, false
    }
    return nil, true
}


/*
 * Pull the managedUrl (external API URL) from the API configuration
 */
func getManagedUrl(api *whisk.RetApi, relpath string, operation string) (url string) {
    baseUrl := strings.TrimSuffix(api.BaseUrl, "/")
    whisk.Debug(whisk.DbgInfo, "getManagedUrl: baseUrl = %s, relpath = %s, operation = %s\n", baseUrl, relpath, operation)
    for path, _ := range api.Swagger.Paths {
        whisk.Debug(whisk.DbgInfo, "getManagedUrl: comparing api relpath: %s\n", path)
        if (path == relpath) {
            whisk.Debug(whisk.DbgInfo, "getManagedUrl: relpath matches '%s'\n", relpath)
            for op, _  := range api.Swagger.Paths[path] {
                whisk.Debug(whisk.DbgInfo, "getManagedUrl: comparing operation: '%s'\n", op)
                if (strings.ToLower(op) == strings.ToLower(operation)) {
                    whisk.Debug(whisk.DbgInfo, "getManagedUrl: operation matches: %s\n", operation)
                    url = baseUrl+path
                }
            }
        }
    }
    return url
}

/////////////
// V2 Cmds //
/////////////
var apiCreateCmdV2 = &cobra.Command{
    Use:           "create ([BASE_PATH] API_PATH API_VERB ACTION] | --config-file CFG_FILE) ",
    Short:         wski18n.T("COMING SOON - create a new API"),
    SilenceUsage:  true,
    SilenceErrors: true,
    PreRunE:       setupClientConfig,
    RunE: func(cmd *cobra.Command, args []string) error {

        var api *whisk.Api
        var err error
        var qname *QualifiedName

        if (!hasApiGwAccessToken()) {
            whisk.Debug(whisk.DbgError, "No APIGW_ACCESS_TOKEN in properties file\n")
            errMsg := wski18n.T("You must login prior to issuing this command.")
            whiskErr := whisk.MakeWskError(errors.New(errMsg), whisk.EXITCODE_ERR_GENERAL,
                whisk.DISPLAY_MSG, whisk.DISPLAY_USAGE)
            return whiskErr
        }

        if (len(args) == 0 && flags.api.configfile == "") {
            whisk.Debug(whisk.DbgError, "No swagger file and no arguments\n")
            errMsg := wski18n.T("Invalid argument(s). Specify a swagger file or specify an API base path with an API path, an API verb, and an action name.")
            whiskErr := whisk.MakeWskError(errors.New(errMsg), whisk.EXITCODE_ERR_GENERAL,
                whisk.DISPLAY_MSG, whisk.DISPLAY_USAGE)
            return whiskErr
        } else if (len(args) == 0 && flags.api.configfile != "") {
            api, err = parseSwaggerApi()
            if err != nil {
                whisk.Debug(whisk.DbgError, "parseSwaggerApi() error: %s\n", err)
                errMsg := wski18n.T("Unable to parse swagger file: {{.err}}", map[string]interface{}{"err": err})
                whiskErr := whisk.MakeWskErrorFromWskError(errors.New(errMsg), err, whisk.EXITCODE_ERR_GENERAL,
                    whisk.DISPLAY_MSG, whisk.DISPLAY_USAGE)
                return whiskErr
            }
        } else {
            if whiskErr := checkArgs(args, 3, 4, "Api create",
                wski18n.T("Specify a swagger file or specify an API base path with an API path, an API verb, and an action name.")); whiskErr != nil {
                return whiskErr
            }
            api, qname, err = parseApiV2(cmd, args)
            if err != nil {
                whisk.Debug(whisk.DbgError, "parseApiV2(%s, %s) error: %s\n", cmd, args, err)
                errMsg := wski18n.T("Unable to parse api command arguments: {{.err}}",
                    map[string]interface{}{"err": err})
                whiskErr := whisk.MakeWskErrorFromWskError(errors.New(errMsg), err, whisk.EXITCODE_ERR_GENERAL,
                    whisk.DISPLAY_MSG, whisk.DISPLAY_USAGE)
                return whiskErr
            }

            // Confirm that the specified action is a web-action
            if ok, errMsg := isWebAction(client, *qname); !ok {
                whisk.Debug(whisk.DbgError, "isWebAction(%v) is false\n", qname)
                whiskErr := whisk.MakeWskErrorFromWskError(errors.New(errMsg), err, whisk.EXITCODE_ERR_GENERAL,
                    whisk.DISPLAY_MSG, whisk.DISPLAY_USAGE)
                return whiskErr
            }
        }

        apiCreateReq := new(whisk.ApiCreateRequest)
        apiCreateReq.ApiDoc = api

        apiCreateReqOptions := new(whisk.ApiCreateRequestOptions)
        props, _ := readProps(Properties.PropsFile)
        apiCreateReqOptions.AccessToken = props["APIGW_ACCESS_TOKEN"]
        apiCreateReqOptions.SpaceGuid = strings.Split(props["AUTH"], ":")[0]
        whisk.Debug(whisk.DbgInfo, "AccessToken: %s\nSpaceGuid: %s\n", apiCreateReqOptions.AccessToken, apiCreateReqOptions.SpaceGuid)

        retApi, _, err := client.Apis.InsertV2(apiCreateReq, apiCreateReqOptions, whisk.DoNotOverwrite)
        if err != nil {
            whisk.Debug(whisk.DbgError, "client.Apis.InsertV2(%#v, false) error: %s\n", api, err)
            errMsg := wski18n.T("Unable to create API: {{.err}}", map[string]interface{}{"err": err})
            whiskErr := whisk.MakeWskErrorFromWskError(errors.New(errMsg), err, whisk.EXITCODE_ERR_NETWORK,
                whisk.DISPLAY_MSG, whisk.NO_DISPLAY_USAGE)
            return whiskErr
        }

        if (api.Swagger == "") {
            baseUrl := retApi.BaseUrl
            fmt.Fprintf(color.Output,
                wski18n.T("{{.ok}} created API {{.path}} {{.verb}} for action {{.name}}\n{{.fullpath}}\n",
                    map[string]interface{}{
                        "ok": color.GreenString("ok:"),
                        "path": strings.TrimSuffix(api.GatewayBasePath, "/")+api.GatewayRelPath,
                        "verb": api.GatewayMethod,
                        "name": boldString("/"+api.Action.Namespace+"/"+api.Action.Name),
                        "fullpath": strings.TrimSuffix(baseUrl, "/")+api.GatewayRelPath,
                    }))
        } else {
            whisk.Debug(whisk.DbgInfo, "Processing swagger based create API response\n")
            baseUrl := retApi.BaseUrl
            for path, _ := range retApi.Swagger.Paths {
                managedUrl := strings.TrimSuffix(baseUrl, "/")+path
                whisk.Debug(whisk.DbgInfo, "Managed path: %s\n",managedUrl)
                for op, opv  := range retApi.Swagger.Paths[path] {
                    whisk.Debug(whisk.DbgInfo, "Path operation: %s\n", op)
                    var fqActionName string
                    if (len(opv.XOpenWhisk.Package) > 0) {
                        fqActionName = "/"+opv.XOpenWhisk.Namespace+"/"+opv.XOpenWhisk.Package+"/"+opv.XOpenWhisk.ActionName
                    } else {
                        fqActionName = "/"+opv.XOpenWhisk.Namespace+"/"+opv.XOpenWhisk.ActionName
                    }
                    whisk.Debug(whisk.DbgInfo, "baseUrl %s  Path %s  Path obj %+v\n", baseUrl, path, opv)
                    fmt.Fprintf(color.Output,
                        wski18n.T("{{.ok}} created API {{.path}} {{.verb}} for action {{.name}}\n{{.fullpath}}\n",
                            map[string]interface{}{
                                "ok": color.GreenString("ok:"),
                                "path": strings.TrimSuffix(retApi.Swagger.BasePath, "/") + path,
                                "verb": op,
                                "name": boldString(fqActionName),
                                "fullpath": managedUrl,
                            }))
                }
            }
        }


        return nil
    },
}

var apiGetCmdV2 = &cobra.Command{
    Use:           "get BASE_PATH | API_NAME",
    Short:         wski18n.T("COMING SOON - get API details"),
    SilenceUsage:  true,
    SilenceErrors: true,
    PreRunE:       setupClientConfig,
    RunE: func(cmd *cobra.Command, args []string) error {
        var err error
        var isBasePathArg bool = true

        if (!hasApiGwAccessToken()) {
            whisk.Debug(whisk.DbgError, "No APIGW_ACCESS_TOKEN in properties file\n")
            errMsg := wski18n.T("You must login prior to issuing this command.")
            whiskErr := whisk.MakeWskError(errors.New(errMsg), whisk.EXITCODE_ERR_GENERAL,
                whisk.DISPLAY_MSG, whisk.DISPLAY_USAGE)
            return whiskErr
        }

        if whiskErr := checkArgs(args, 1, 1, "Api get",
            wski18n.T("An API base path or API name is required.")); whiskErr != nil {
            return whiskErr
        }

        apiGetReq := new(whisk.ApiGetRequest)
        apiGetReqOptions := new(whisk.ApiGetRequestOptions)
        apiGetReqOptions.ApiBasePath = args[0]
        props, _ := readProps(Properties.PropsFile)
        apiGetReqOptions.AccessToken = props["APIGW_ACCESS_TOKEN"]
        apiGetReqOptions.SpaceGuid = strings.Split(props["AUTH"], ":")[0]


        retApi, _, err := client.Apis.GetV2(apiGetReq, apiGetReqOptions)
        if err != nil {
            whisk.Debug(whisk.DbgError, "client.Apis.GetV2(%#v, %#v) error: %s\n", apiGetReq, apiGetReqOptions, err)
            errMsg := wski18n.T("Unable to get API '{{.name}}': {{.err}}", map[string]interface{}{"name": args[0], "err": err})
            whiskErr := whisk.MakeWskErrorFromWskError(errors.New(errMsg), err, whisk.EXITCODE_ERR_GENERAL,
                whisk.DISPLAY_MSG, whisk.NO_DISPLAY_USAGE)
            return whiskErr
        }
        whisk.Debug(whisk.DbgInfo, "client.Apis.GetV2 returned: %#v\n", retApi)

        var displayResult interface{} = nil
        if (flags.common.detail) {
            if (retApi.Apis != nil && len(retApi.Apis) > 0 &&
            retApi.Apis[0].ApiValue != nil) {
                displayResult = retApi.Apis[0].ApiValue
            } else {
                whisk.Debug(whisk.DbgError, "No result object returned\n")
            }
        } else {
            if (retApi.Apis != nil && len(retApi.Apis) > 0 &&
            retApi.Apis[0].ApiValue != nil &&
            retApi.Apis[0].ApiValue.Swagger != nil) {
                displayResult = retApi.Apis[0].ApiValue.Swagger
            } else {
                whisk.Debug(whisk.DbgError, "No swagger returned\n")
            }
        }
        if (displayResult == nil) {
            var errMsg string
            if (isBasePathArg) {
                errMsg = wski18n.T("API does not exist for basepath {{.basepath}}",
                    map[string]interface{}{"basepath": args[0]})
            } else {
                errMsg = wski18n.T("API does not exist for API name {{.apiname}}",
                    map[string]interface{}{"apiname": args[0]})
            }

            whiskErr := whisk.MakeWskError(errors.New(errMsg), whisk.EXITCODE_ERR_GENERAL,
                whisk.DISPLAY_MSG, whisk.NO_DISPLAY_USAGE)
            return whiskErr
        }
        printJSON(displayResult)

        return nil
    },
}

var apiDeleteCmdV2 = &cobra.Command{
    Use:           "delete BASE_PATH | API_NAME [API_PATH [API_VERB]]",
    Short:         wski18n.T("COMING SOON - delete an API"),
    SilenceUsage:  true,
    SilenceErrors: true,
    PreRunE:       setupClientConfig,
    RunE: func(cmd *cobra.Command, args []string) error {

        if (!hasApiGwAccessToken()) {
            whisk.Debug(whisk.DbgError, "No APIGW_ACCESS_TOKEN in properties file\n")
            errMsg := wski18n.T("You must login prior to issuing this command.")
            whiskErr := whisk.MakeWskError(errors.New(errMsg), whisk.EXITCODE_ERR_GENERAL,
                whisk.DISPLAY_MSG, whisk.DISPLAY_USAGE)
            return whiskErr
        }

        if whiskErr := checkArgs(args, 1, 3, "Api delete",
            wski18n.T("An API base path or API name is required.  An optional API relative path and operation may also be provided.")); whiskErr != nil {
            return whiskErr
        }

        apiDeleteReq := new(whisk.ApiDeleteRequest)
        apiDeleteReqOptions := new(whisk.ApiDeleteRequestOptions)
        props, _ := readProps(Properties.PropsFile)
        apiDeleteReqOptions.AccessToken = props["APIGW_ACCESS_TOKEN"]
        apiDeleteReqOptions.SpaceGuid = strings.Split(props["AUTH"], ":")[0]

        // Is the argument a basepath (must start with /) or an API name
        if _, ok := isValidBasepath(args[0]); !ok {
            whisk.Debug(whisk.DbgInfo, "Treating '%s' as an API name; as it does not begin with '/'\n", args[0])
            apiDeleteReqOptions.ApiBasePath = args[0]
        } else {
            apiDeleteReqOptions.ApiBasePath = args[0]
        }

        if (len(args) > 1) {
            // Is the API path valid?
            if whiskErr, ok := isValidRelpath(args[1]); !ok {
                return whiskErr
            }
            apiDeleteReqOptions.ApiRelPath = args[1]
        }
        if (len(args) > 2) {
            // Is the API verb valid?
            if whiskErr, ok := IsValidApiVerb(args[2]); !ok {
                return whiskErr
            }
            apiDeleteReqOptions.ApiVerb = strings.ToUpper(args[2])
        }

        _, err := client.Apis.DeleteV2(apiDeleteReq, apiDeleteReqOptions)
        if err != nil {
            whisk.Debug(whisk.DbgError, "client.Apis.DeleteV2(%#v, %#v) error: %s\n", apiDeleteReq, apiDeleteReqOptions, err)
            errMsg := wski18n.T("Unable to delete action: {{.err}}", map[string]interface{}{"err": err})
            whiskErr := whisk.MakeWskErrorFromWskError(errors.New(errMsg), err, whisk.EXITCODE_ERR_GENERAL,
                whisk.DISPLAY_MSG, whisk.NO_DISPLAY_USAGE)
            return whiskErr
        }

        if (len(args) == 1) {
            fmt.Fprintf(color.Output,
                wski18n.T("{{.ok}} deleted API {{.basepath}}\n",
                    map[string]interface{}{
                        "ok": color.GreenString("ok:"),
                        "basepath": apiDeleteReqOptions.ApiBasePath,
                    }))
        } else if (len(args) == 2 ) {
            fmt.Fprintf(color.Output,
                wski18n.T("{{.ok}} deleted {{.path}} from {{.basepath}}\n",
                    map[string]interface{}{
                        "ok": color.GreenString("ok:"),
                        "path": apiDeleteReqOptions.ApiRelPath,
                        "basepath": apiDeleteReqOptions.ApiBasePath,
                    }))
        } else {
            fmt.Fprintf(color.Output,
                wski18n.T("{{.ok}} deleted {{.path}} {{.verb}} from {{.basepath}}\n",
                    map[string]interface{}{
                        "ok": color.GreenString("ok:"),
                        "path": apiDeleteReqOptions.ApiRelPath,
                        "verb": apiDeleteReqOptions.ApiVerb,
                        "basepath": apiDeleteReqOptions.ApiBasePath,
                    }))
        }

        return nil
    },
}

var apiListCmdV2 = &cobra.Command{
    Use:           "list [[BASE_PATH | API_NAME] [API_PATH [API_VERB]]",
    Short:         wski18n.T("COMING SOON - list APIs"),
    SilenceUsage:  true,
    SilenceErrors: true,
    PreRunE:       setupClientConfig,
    RunE: func(cmd *cobra.Command, args []string) error {
        var err error
        var retApiList *whisk.ApiListResponseV2
        var retApi *whisk.ApiGetResponseV2
        var retApiArray *whisk.RetApiArrayV2

        if (!hasApiGwAccessToken()) {
            whisk.Debug(whisk.DbgError, "No APIGW_ACCESS_TOKEN in properties file\n")
            errMsg := wski18n.T("You must login prior to issuing this command.")
            whiskErr := whisk.MakeWskError(errors.New(errMsg), whisk.EXITCODE_ERR_GENERAL,
                whisk.DISPLAY_MSG, whisk.DISPLAY_USAGE)
            return whiskErr
        }

        if whiskErr := checkArgs(args, 0, 3, "Api list",
            wski18n.T("Optional parameters are: API base path (or API name), API relative path and operation.")); whiskErr != nil {
            return whiskErr
        }

        props, _ := readProps(Properties.PropsFile)
        accesstoken := props["APIGW_ACCESS_TOKEN"]
        spaceguid := strings.Split(props["AUTH"], ":")[0]

        // Get API request body
        apiGetReq := new(whisk.ApiGetRequest)
        apiGetReq.Namespace = client.Config.Namespace
        // Get API request options
        apiGetReqOptions := new(whisk.ApiGetRequestOptions)
        apiGetReqOptions.AccessToken = accesstoken
        apiGetReqOptions.SpaceGuid = spaceguid

        // List API request query parameters
        apiListReqOptions := new(whisk.ApiListRequestOptions)
        apiListReqOptions.Limit = flags.common.limit
        apiListReqOptions.Skip = flags.common.skip
        apiListReqOptions.AccessToken = accesstoken
        apiListReqOptions.SpaceGuid = spaceguid

        if (len(args) == 0) {
            retApiList, _, err = client.Apis.ListV2(apiListReqOptions)
            if err != nil {
                whisk.Debug(whisk.DbgError, "client.Apis.ListV2(%#v) error: %s\n", apiListReqOptions, err)
                errMsg := wski18n.T("Unable to obtain the API list: {{.err}}", map[string]interface{}{"err": err})
                whiskErr := whisk.MakeWskErrorFromWskError(errors.New(errMsg), err, whisk.EXITCODE_ERR_GENERAL,
                    whisk.DISPLAY_MSG, whisk.NO_DISPLAY_USAGE)
                return whiskErr
            }
            whisk.Debug(whisk.DbgInfo, "client.Apis.ListV2 returned: %#v (%+v)\n", retApiList, retApiList)
            // Cast to a common type to allow for code to print out apilist response or apiget response
            retApiArray = (*whisk.RetApiArrayV2)(retApiList)
        } else {
            // The first argument is either a basepath (must start with /) or an API name
            apiGetReqOptions.ApiBasePath = args[0]
            if (len(args) > 1) {
                // Is the API path valid?
                if whiskErr, ok := isValidRelpath(args[1]); !ok {
                    return whiskErr
                }
                apiGetReqOptions.ApiRelPath = args[1]
            }
            if (len(args) > 2) {
                // Is the API verb valid?
                if whiskErr, ok := IsValidApiVerb(args[2]); !ok {
                    return whiskErr
                }
                apiGetReqOptions.ApiVerb = strings.ToUpper(args[2])
            }

            retApi, _, err = client.Apis.GetV2(apiGetReq, apiGetReqOptions)
            if err != nil {
                whisk.Debug(whisk.DbgError, "client.Apis.GetV2(%#v, %#v) error: %s\n", apiGetReq, apiGetReqOptions, err)
                errMsg := wski18n.T("Unable to obtain the API list: {{.err}}", map[string]interface{}{"err": err})
                whiskErr := whisk.MakeWskErrorFromWskError(errors.New(errMsg), err, whisk.EXITCODE_ERR_GENERAL,
                    whisk.DISPLAY_MSG, whisk.NO_DISPLAY_USAGE)
                return whiskErr
            }
            whisk.Debug(whisk.DbgInfo, "client.Apis.GetV2 returned: %#v\n", retApi)
            // Cast to a common type to allow for code to print out apilist response or apiget response
            retApiArray = (*whisk.RetApiArrayV2)(retApi)
        }

        // Display the APIs - applying any specified filtering
        if (flags.common.full) {
            fmt.Fprintf(color.Output,
                wski18n.T("{{.ok}} APIs\n",
                    map[string]interface{}{
                        "ok": color.GreenString("ok:"),
                    }))

            for i:=0; i<len(retApiArray.Apis); i++ {
                printFilteredListApiV2(retApiArray.Apis[i].ApiValue, (*whisk.ApiOptions)(apiGetReqOptions))
            }
        } else {
            if (len(retApiArray.Apis) > 0) {
                // Dynamically create the output format string based on the maximum size of the
                // fully qualified action name and the API Name.
                maxActionNameSize := min(40, max(len("Action"), getLargestActionNameSizeV2(retApiArray, (*whisk.ApiOptions)(apiGetReqOptions))))
                maxApiNameSize := min(30, max(len("API Name"), getLargestApiNameSizeV2(retApiArray, (*whisk.ApiOptions)(apiGetReqOptions))))
                fmtString = "%-"+strconv.Itoa(maxActionNameSize)+"s %7s %"+strconv.Itoa(maxApiNameSize+1)+"s  %s\n"
                fmt.Fprintf(color.Output,
                    wski18n.T("{{.ok}} APIs\n",
                        map[string]interface{}{
                            "ok": color.GreenString("ok:"),
                        }))
                fmt.Printf(fmtString, "Action", "Verb", "API Name", "URL")
                for i:=0; i<len(retApiArray.Apis); i++ {
                    printFilteredListRowV2(retApiArray.Apis[i].ApiValue, (*whisk.ApiOptions)(apiGetReqOptions), maxActionNameSize, maxApiNameSize)
                }
            } else {
                fmt.Fprintf(color.Output,
                    wski18n.T("{{.ok}} APIs\n",
                        map[string]interface{}{
                            "ok": color.GreenString("ok:"),
                        }))
                fmt.Printf(fmtString, "Action", "Verb", "API Name", "URL")
            }
        }

        return nil
    },
}

/*
 * Takes an API object (containing one more more single basepath/relpath/operation triplets)
 * and some filtering configuration.  For each API endpoint matching the filtering criteria, display
 * each endpoint's configuration - one line per configuration property (action name, verb, api name, api gw url)
 */
func printFilteredListApiV2(resultApi *whisk.RetApiV2, api *whisk.ApiOptions) {
    baseUrl := strings.TrimSuffix(resultApi.BaseUrl, "/")
    apiName := resultApi.Swagger.Info.Title
    basePath := resultApi.Swagger.BasePath
    if (resultApi.Swagger != nil && resultApi.Swagger.Paths != nil) {
        for path, _ := range resultApi.Swagger.Paths {
            whisk.Debug(whisk.DbgInfo, "printFilteredListApiV2: comparing api relpath: %s\n", path)
            if ( len(api.ApiRelPath) == 0 || path == api.ApiRelPath) {
                whisk.Debug(whisk.DbgInfo, "printFilteredListApiV2: relpath matches\n")
                for op, opv  := range resultApi.Swagger.Paths[path] {
                    whisk.Debug(whisk.DbgInfo, "printFilteredListApiV2: comparing operation: '%s'\n", op)
                    if ( len(api.ApiVerb) == 0 || strings.ToLower(op) == strings.ToLower(api.ApiVerb)) {
                        whisk.Debug(whisk.DbgInfo, "printFilteredListApiV2: operation matches: %#v\n", opv)
                        var actionName string
                        if (len(opv.XOpenWhisk.Package) > 0) {
                            actionName = "/"+opv.XOpenWhisk.Namespace+"/"+opv.XOpenWhisk.Package+"/"+opv.XOpenWhisk.ActionName
                        } else {
                            actionName = "/"+opv.XOpenWhisk.Namespace+"/"+opv.XOpenWhisk.ActionName
                        }
                        fmt.Printf("%s: %s\n", wski18n.T("Action"), actionName)
                        fmt.Printf("  %s: %s\n", wski18n.T("API Name"), apiName)
                        fmt.Printf("  %s: %s\n", wski18n.T("Base path"), basePath)
                        fmt.Printf("  %s: %s\n", wski18n.T("Path"), path)
                        fmt.Printf("  %s: %s\n", wski18n.T("Verb"), op)
                        fmt.Printf("  %s: %s\n", wski18n.T("URL"), baseUrl+path)
                    }
                }
            }
        }
    }
}

/*
 * Takes an API object (containing one more more single basepath/relpath/operation triplets)
 * and some filtering configuration.  For each API matching the filtering criteria, display the API
 * on a single line (action name, verb, api name, api gw url).
 *
 * NOTE: Large action name and api name value will be truncated by their associated max size parameters.
 */
func printFilteredListRowV2(resultApi *whisk.RetApiV2, api *whisk.ApiOptions, maxActionNameSize int, maxApiNameSize int) {
    baseUrl := strings.TrimSuffix(resultApi.BaseUrl, "/")
    apiName := resultApi.Swagger.Info.Title
    if (resultApi.Swagger != nil && resultApi.Swagger.Paths != nil) {
        for path, _ := range resultApi.Swagger.Paths {
            whisk.Debug(whisk.DbgInfo, "printFilteredListRowV2: comparing api relpath: %s\n", path)
            if ( len(api.ApiRelPath) == 0 || path == api.ApiRelPath) {
                whisk.Debug(whisk.DbgInfo, "printFilteredListRowV2: relpath matches\n")
                for op, opv  := range resultApi.Swagger.Paths[path] {
                    whisk.Debug(whisk.DbgInfo, "printFilteredListRowV2: comparing operation: '%s'\n", op)
                    if ( len(api.ApiVerb) == 0 || strings.ToLower(op) == strings.ToLower(api.ApiVerb)) {
                        whisk.Debug(whisk.DbgInfo, "printFilteredListRowV2: operation matches: %#v\n", opv)
                        var actionName string
                        if (len(opv.XOpenWhisk.Package) > 0) {
                            actionName = "/"+opv.XOpenWhisk.Namespace+"/"+opv.XOpenWhisk.Package+"/"+opv.XOpenWhisk.ActionName
                        } else {
                            actionName = "/"+opv.XOpenWhisk.Namespace+"/"+opv.XOpenWhisk.ActionName
                        }
                        fmt.Printf(fmtString,
                            actionName[0 : min(len(actionName), maxActionNameSize)],
                            op,
                            apiName[0 : min(len(apiName), maxApiNameSize)],
                            baseUrl+path)
                    }
                }
            }
        }
    }
}

func getLargestActionNameSizeV2(retApiArray *whisk.RetApiArrayV2, api *whisk.ApiOptions) int {
    var maxNameSize = 0
    for i:=0; i<len(retApiArray.Apis); i++ {
        var resultApi = retApiArray.Apis[i].ApiValue
        if (resultApi.Swagger != nil && resultApi.Swagger.Paths != nil) {
            for path, _ := range resultApi.Swagger.Paths {
                whisk.Debug(whisk.DbgInfo, "getLargestActionNameSize: comparing api relpath: %s\n", path)
                if ( len(api.ApiRelPath) == 0 || path == api.ApiRelPath) {
                    whisk.Debug(whisk.DbgInfo, "getLargestActionNameSize: relpath matches\n")
                    for op, opv  := range resultApi.Swagger.Paths[path] {
                        whisk.Debug(whisk.DbgInfo, "getLargestActionNameSize: comparing operation: '%s'\n", op)
                        if ( len(api.ApiVerb) == 0 || strings.ToLower(op) == strings.ToLower(api.ApiVerb)) {
                            whisk.Debug(whisk.DbgInfo, "getLargestActionNameSize: operation matches: %#v\n", opv)
                            var fullActionName string
                            if (len(opv.XOpenWhisk.Package) > 0) {
                                fullActionName = "/"+opv.XOpenWhisk.Namespace+"/"+opv.XOpenWhisk.Package+"/"+opv.XOpenWhisk.ActionName
                            } else {
                                fullActionName = "/"+opv.XOpenWhisk.Namespace+"/"+opv.XOpenWhisk.ActionName
                            }
                            if (len(fullActionName) > maxNameSize) {
                                maxNameSize = len(fullActionName)
                            }
                        }
                    }
                }
            }
        }
    }
    return maxNameSize
}

func getLargestApiNameSizeV2(retApiArray *whisk.RetApiArrayV2, api *whisk.ApiOptions) int {
    var maxNameSize = 0
    for i:=0; i<len(retApiArray.Apis); i++ {
        var resultApi = retApiArray.Apis[i].ApiValue
        apiName := resultApi.Swagger.Info.Title
        if (resultApi.Swagger != nil && resultApi.Swagger.Paths != nil) {
            for path, _ := range resultApi.Swagger.Paths {
                whisk.Debug(whisk.DbgInfo, "getLargestActionNameSize: comparing api relpath: %s\n", path)
                if ( len(api.ApiRelPath) == 0 || path == api.ApiRelPath) {
                    whisk.Debug(whisk.DbgInfo, "getLargestActionNameSize: relpath matches\n")
                    for op, opv  := range resultApi.Swagger.Paths[path] {
                        whisk.Debug(whisk.DbgInfo, "getLargestActionNameSize: comparing operation: '%s'\n", op)
                        if ( len(api.ApiVerb) == 0 || strings.ToLower(op) == strings.ToLower(api.ApiVerb)) {
                            whisk.Debug(whisk.DbgInfo, "getLargestActionNameSize: operation matches: %#v\n", opv)
                            if (len(apiName) > maxNameSize) {
                                maxNameSize = len(apiName)
                            }
                        }
                    }
                }
            }
        }
    }
    return maxNameSize
}

func hasApiGwAccessToken() bool {
    props, _ := readProps(Properties.PropsFile)
    return (len(props["APIGW_ACCESS_TOKEN"]) > 0)
}

/*
 * if # args = 4
 * args[0] = API base path
 * args[0] = API relative path
 * args[1] = API verb
 * args[2] = Optional.  Action name (may or may not be qualified with namespace and package name)
 *
 * if # args = 3
 * args[0] = API relative path
 * args[1] = API verb
 * args[2] = Optional.  Action name (may or may not be qualified with namespace and package name)
 */
func parseApiV2(cmd *cobra.Command, args []string) (*whisk.Api, *QualifiedName, error) {
    var err error
    var basepath string = "/"
    var apiname string
    var basepathArgIsApiName = false;

    api := new(whisk.Api)

    if (len(args) > 3) {
        // Is the argument a basepath (must start with /) or an API name
        if _, ok := isValidBasepath(args[0]); !ok {
            whisk.Debug(whisk.DbgInfo, "Treating '%s' as an API name; as it does not begin with '/'\n", args[0])
            basepathArgIsApiName = true;
        }
        basepath = args[0]

        // Shift the args so the remaining code works with or without the explicit base path arg
        args = args[1:]
    }

    // Is the API path valid?
    if (len(args) > 0) {
        if whiskErr, ok := isValidRelpath(args[0]); !ok {
            return nil, nil, whiskErr
        }
        api.GatewayRelPath = args[0]    // Maintain case as URLs may be case-sensitive
    }

    // Is the API verb valid?
    if (len(args) > 1) {
        if whiskErr, ok := IsValidApiVerb(args[1]); !ok {
            return nil, nil, whiskErr
        }
        api.GatewayMethod = strings.ToUpper(args[1])
    }

    // Is the specified action name valid?
    var qName QualifiedName
    if (len(args) == 3) {
        qName = QualifiedName{}
        qName, err = parseQualifiedName(args[2])
        if err != nil {
            whisk.Debug(whisk.DbgError, "parseQualifiedName(%s) failed: %s\n", args[2], err)
            errMsg := wski18n.T("'{{.name}}' is not a valid action name: {{.err}}",
                map[string]interface{}{"name": args[2], "err": err})
            whiskErr := whisk.MakeWskErrorFromWskError(errors.New(errMsg), err, whisk.EXITCODE_ERR_GENERAL,
                whisk.DISPLAY_MSG, whisk.DISPLAY_USAGE)
            return nil, nil, whiskErr
        }
        if (qName.entityName == "") {
            whisk.Debug(whisk.DbgError, "Action name '%s' is invalid\n", args[2])
            errMsg := wski18n.T("'{{.name}}' is not a valid action name.", map[string]interface{}{"name": args[2]})
            whiskErr := whisk.MakeWskErrorFromWskError(errors.New(errMsg), err, whisk.EXITCODE_ERR_GENERAL,
                whisk.DISPLAY_MSG, whisk.DISPLAY_USAGE)
            return nil, nil, whiskErr
        }
    }

    if ( len(flags.api.apiname) > 0 ) {
        if (basepathArgIsApiName) {
            // Specifying API name as argument AND as a --apiname option value is invalid
            whisk.Debug(whisk.DbgError, "API is specified as an argument '%s' and as a flag '%s'\n", basepath, flags.api.apiname)
            errMsg := wski18n.T("An API name can only be specified once.")
            whiskErr := whisk.MakeWskError(errors.New(errMsg), whisk.EXITCODE_ERR_GENERAL,
                whisk.DISPLAY_MSG, whisk.DISPLAY_USAGE)
            return nil, nil, whiskErr
        }
        apiname = flags.api.apiname
    }

    api.Namespace = client.Config.Namespace
    api.Action = new(whisk.ApiAction)
    var urlActionPackage string
    if (len(qName.packageName) > 0) {
        urlActionPackage = qName.packageName
    } else {
        urlActionPackage = "default"
    }
    api.Action.BackendUrl = "https://" + client.Config.Host + "/api/v1/web/" + qName.namespace + "/" + urlActionPackage + "/" + qName.entity + ".http"
    api.Action.BackendMethod = api.GatewayMethod
    api.Action.Name = qName.entityName
    api.Action.Namespace = qName.namespace
    api.Action.Auth = client.Config.AuthToken
    api.ApiName = apiname
    api.GatewayBasePath = basepath
    if (!basepathArgIsApiName) { api.Id = "API:"+api.Namespace+":"+api.GatewayBasePath }

    whisk.Debug(whisk.DbgInfo, "Parsed api struct: %#v\n", api)
    return api, &qName, nil
}

///////////
// Flags //
///////////

func init() {
    apiCreateCmd.Flags().StringVarP(&flags.api.apiname, "apiname", "n", "", wski18n.T("Friendly name of the API; ignored when CFG_FILE is specified (default BASE_PATH)"))
    apiCreateCmd.Flags().StringVarP(&flags.api.configfile, "config-file", "c", "", wski18n.T("`CFG_FILE` containing API configuration in swagger JSON format"))
    //apiUpdateCmd.Flags().StringVarP(&flags.api.action, "action", "a", "", wski18n.T("`ACTION` to invoke when API is called"))
    //apiUpdateCmd.Flags().StringVarP(&flags.api.path, "path", "p", "", wski18n.T("relative `PATH` of API"))
    //apiUpdateCmd.Flags().StringVarP(&flags.api.verb, "method", "m", "", wski18n.T("API `VERB`"))
    apiGetCmd.Flags().BoolVarP(&flags.common.detail, "full", "f", false, wski18n.T("display full API configuration details"))
    apiListCmd.Flags().IntVarP(&flags.common.skip, "skip", "s", 0, wski18n.T("exclude the first `SKIP` number of actions from the result"))
    apiListCmd.Flags().IntVarP(&flags.common.limit, "limit", "l", 30, wski18n.T("only return `LIMIT` number of actions from the collection"))
    apiListCmd.Flags().BoolVarP(&flags.common.full, "full", "f", false, wski18n.T("display full description of each API"))
    apiExperimentalCmd.AddCommand(
        apiCreateCmd,
        //apiUpdateCmd,
        apiGetCmd,
        apiDeleteCmd,
        apiListCmd,
    )

    apiCreateCmdV2.Flags().StringVarP(&flags.api.apiname, "apiname", "n", "", wski18n.T("Friendly name of the API; ignored when CFG_FILE is specified (default BASE_PATH)"))
    apiCreateCmdV2.Flags().StringVarP(&flags.api.configfile, "config-file", "c", "", wski18n.T("`CFG_FILE` containing API configuration in swagger JSON format"))
    apiGetCmdV2.Flags().BoolVarP(&flags.common.detail, "full", "f", false, wski18n.T("display full API configuration details"))
    apiListCmdV2.Flags().IntVarP(&flags.common.skip, "skip", "s", 0, wski18n.T("exclude the first `SKIP` number of actions from the result"))
    apiListCmdV2.Flags().IntVarP(&flags.common.limit, "limit", "l", 30, wski18n.T("only return `LIMIT` number of actions from the collection"))
    apiListCmdV2.Flags().BoolVarP(&flags.common.full, "full", "f", false, wski18n.T("display full description of each API"))
    apiCmd.AddCommand(
        apiCreateCmdV2,
        apiGetCmdV2,
        apiDeleteCmdV2,
        apiListCmdV2,
    )
}
