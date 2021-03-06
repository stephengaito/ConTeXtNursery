// Copyright 2020 PerceptiSys Ltd, (Stephen Gaito)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// A webserver which implements hierachical prefix routing of RESTful
// HTTP interfaces.
//
package webserver

import (
  "bytes"
  "crypto/tls"
  "crypto/x509"
  "encoding/json"
  "fmt"
  "github.com/diSimplex/ConTeXtNursery/logger"
  "io"
  "io/ioutil"
  "net"
  "net/http"
  "strings"
  "time"
)

//////////////////////////////////////////////////////////////////////
// Webserver Types
//

type Route struct {
  Prefix        string
  Path          string
  Desc          string
  Visible       bool
  SubRoutes     []*Route
  GetHandler    http.HandlerFunc
  HeadHandler   http.HandlerFunc
  PostHandler   http.HandlerFunc
  PutHandler    http.HandlerFunc
  DeleteHandler http.HandlerFunc
}

type RouteDesc struct {
  Prefix        string
  Path          string
  Desc          string
  Visible       bool
  SubRoutes     []*RouteDesc
}

type WS struct {
  Listener   net.Listener
  Server    *http.Server
  HostPort   string
  BaseRoute *Route
  InitTime   time.Time
  Log       *logger.LoggerType
}

//////////////////////////////////////////////////////////////////////
// Webserver functions
//

// Create a webserver with no routes listening on the given host and
// port using the given tls.Config.
//
func CreateWebServer(
  host, port, description string,
  caCertPath, certPath, keyPath string,
  cnLog     *logger.LoggerType,
) *WS {
  var err error

  // Load the Server x509 Certificates and keys for this webServer
  //
  serverCert, err := tls.LoadX509KeyPair( certPath, keyPath )
  cnLog.MayBeFatal("Could not load cert/key pair", err)
  //
  caCert, err := ioutil.ReadFile(caCertPath)
  cnLog.MayBeFatal("Could not load the CA certificate", err)
  //
  caCertPool := x509.NewCertPool()
  caCertPool.AppendCertsFromPEM(caCert)
  //
  // Setup HTTPS server configuration
  //
  tlsConfig := &tls.Config{
    ClientAuth:     tls.RequireAndVerifyClientCert,
    Certificates: []tls.Certificate{serverCert},
    RootCAs:        caCertPool,
    ClientCAs:      caCertPool,
  }

  // now create the WebServer structure itself
  //
  ws          := WS{}
  ws.InitTime  = time.Now()
  ws.Log       = cnLog
  ws.BaseRoute = ws.CreateNewRoute("/", "", description, true)
  ws.HostPort  = host + ":" + port
  ws.Log.Logf("listening at [%s]\n", ws.HostPort)
  ws.Listener, err = tls.Listen("tcp",  ws.HostPort, tlsConfig)
  ws.Log.MayBeFatal("Could not create listener", err)

  ws.Server = &http.Server{
    Handler:   &ws,
    TLSConfig: tlsConfig,
  }

  return &ws
}

// Reply in JSON marshaled from the given value.
//
func (ws *WS) ReplyInJson(
  w http.ResponseWriter,
  r *http.Request,
  value interface{},
) {
  jsonBytes, err := json.Marshal(value)
  if err != nil {
    ws.Log.MayBeError("Could not json.marshal value in repliedInJson", err)
    jsonBytes = []byte{}
  }
  w.Write(jsonBytes)
}

func (ws *WS) ReplyAsRawFile(
  w http.ResponseWriter,
  r *http.Request,
  file io.Reader,
  mimeType string,
) bool {
  // TODO
  return false
}

func (ws *WS) ReadLines(file io.Reader) []string {
  // TODO
  return []string{}
}

// Helper method to parse the path parts in a multi-part url.
//
// CALLS ws.Log;
// THREAD-SAFE;
//
func (ws *WS) GetPathParts(aPath string) []string {
  pathParts := strings.Split(strings.TrimPrefix(aPath, "/"), "/")
  ws.Log.Logf(
    "pathParts [%s] len: %d",
    strings.Join(pathParts, "|"),
    len(pathParts),
  )
  return pathParts
}

// The data required to describe how much of a partial route has been found
// by the FindRoute function.
//
type PartialRouteError struct {
  NumPartsFound int
  NumParts      int
  CurPrefix     string
  Message       string
}

// Create a PartialRouteError by providing information about how much of
// the route we *have* found.
//
func CreatePartialRouteError(
  numPartsFound, numParts int,
  curPrefix, message string,
) *PartialRouteError {
  return &PartialRouteError{
    NumPartsFound: numPartsFound,
    NumParts:      numParts,
    CurPrefix:     curPrefix,
    Message:       message,
  }
}

// Provide the standard error interface for PartialRouteErrors
//
func (pr *PartialRouteError) Error() string {
  return pr.Message
}

// Find a route by walking down the hierarchy of routes known to the
// webserver, starting at the BaseRoute.
//
func (ws *WS) FindRoute(url string) (*Route, *PartialRouteError) {
  curRoute := ws.BaseRoute
  if curRoute == nil { return nil, nil } // signal that we need to add a baseRoute
  if url == "/" { return curRoute, nil } // This IS the base route
  
  urlParts := strings.Split(strings.TrimPrefix(url, "/"), "/")
  for urlPartNum, aUrlPart := range urlParts {
    foundRoute := false
    for _, aRoute := range curRoute.SubRoutes {
      if aUrlPart == aRoute.Prefix {
        curRoute = aRoute
        foundRoute = true
        break
      }
    }
    if !foundRoute {
      theErr := CreatePartialRouteError(
        urlPartNum, len(urlParts),
        aUrlPart,
        fmt.Sprintf(
          "Could not find route for [/%s]",
          strings.Join(urlParts[0:urlPartNum+1], "/"),
        ),
      )
      return curRoute, theErr
    }
  }

  return curRoute, nil
}

func CreateRouteDesc(aRoute *Route) *RouteDesc {
  routeDesc := &RouteDesc{
    Desc:      aRoute.Desc,
    Path:      aRoute.Path,
    Prefix:    aRoute.Prefix,
    Visible:   aRoute.Visible,
  }
    
  for _, aSubRoute := range aRoute.SubRoutes {
    routeDesc.SubRoutes = append(routeDesc.SubRoutes, &RouteDesc{
      Desc:    aSubRoute.Desc,
      Path:    aSubRoute.Path,
      Prefix:  aSubRoute.Prefix,
      Visible: aSubRoute.Visible,
    })
  }
  
  return routeDesc
}

func (ws *WS) CreateNewRoute(
  url, prefix, description string,
  visible bool,
) *Route {
  aNewRoute := &Route{
    Desc:    description,
    Path:    url,
    Prefix:  prefix,
    Visible: visible,
  }

  aNewRoute.GetHandler = func (w http.ResponseWriter, r *http.Request) {
    ws.ReplyInJson(w, r, CreateRouteDesc(aNewRoute))
    
//    if ws.RepliedInJson(w, r, CreateRouteDesc(aNewRoute)) { return }
//
//    rdTemplate := ws.RouteDescriptionTemplate()
//    err := rdTemplate.Execute(w, aNewRoute)
//    if err != nil {
//      ws.Log.MayBeError("Could not execute base page template", err)
//      w.Write([]byte("Could not provide any ConTeXt Nursery information\nPlease try again!"))
//    }
  }
  return aNewRoute
}

// Create and describe a route in the webserver.
//
// Returns an error if the route already exists, OR if a parent path does
// not exist.
//
// The route is marked as "Visible" if visible is true.
//
func (ws *WS) DescribeRoute(url, description string, visible bool) error {

  aRoute, err := ws.FindRoute(url)
  if err == nil && aRoute.Path == url {
    return fmt.Errorf("The route [%s] already exists", url)
  }

  if err.NumPartsFound+1 == err.NumParts {
    aNewRoute := ws.CreateNewRoute(url, err.CurPrefix, description, visible)
    aRoute.SubRoutes     = append(aRoute.SubRoutes, aNewRoute)
    return nil
  }

  return err
}

/*
// The default route description template as used by all described routes'
// default GetHandler (unless explicitly over-ridden).
//
func (ws *WS) RouteDescriptionTemplate() *template.Template {
  rdTemplateStr := `
  <body>
    <h1>ConTeXt Nursery on {{.Path}}</h1>
    <ul>
{{ range .SubRoutes }}
{{   if .Visible }}
      <li>
        <strong><a href="{{.Path}}">{{.Path}}</a></strong>
        <p>{{.Desc}}</p>
      </li>
{{   end }}
{{ end }}
    </ul>
 </body>

`
  theTemplate := template.New("body")

  theTemplate, err := theTemplate.Parse(rdTemplateStr)
  ws.Log.MayBeFatal("Could not parse the internal route desciption template", err)

  return theTemplate
}
*/

// Add a http.Handler for http.MethodGet request at the url route.
//
// NOTE: the route must already be described using the DescribeRoute 
// function.
//
func (ws *WS) AddGetHandler(url string, handlerFunc http.HandlerFunc) error {
  aRoute, err := ws.FindRoute(url)
  if err != nil { return err }
  aRoute.GetHandler = handlerFunc
  return nil
}

// Add a http.Handler for http.MethodHead request at the url route.
//
// NOTE: the route must already be described using the DescribeRoute 
// function.
//
func (ws *WS) AddHeadHandler(url string, handlerFunc http.HandlerFunc) error {
  aRoute, err := ws.FindRoute(url)
  if err != nil { return err }
  aRoute.HeadHandler = handlerFunc
  return nil
}

// Add a http.HandlerFunc for http.MethodPost request at the url route.
//
// NOTE: the route must already be described using the DescribeRoute 
// function.
//
func (ws *WS) AddPostHandler(url string, handlerFunc http.HandlerFunc) error {
  aRoute, err := ws.FindRoute(url)
  if err != nil { return err }
  aRoute.PostHandler = handlerFunc
  return nil
}

// Add a http.HandlerFunc for http.MethodPut request at the url route.
//
// NOTE: the route must already be described using the DescribeRoute 
// function.
//
func (ws *WS) AddPutHandler(url string, handlerFunc http.HandlerFunc) error {
  aRoute, err := ws.FindRoute(url)
  if err != nil { return err }
  aRoute.PutHandler = handlerFunc
  return nil
}

// Add a http.HandlerFunc for http.MethodDelete request at the url route.
//
// NOTE: the route must already be described using the DescribeRoute 
// function.
//
func (ws *WS) AddDeleteHandler(url string, handlerFunc http.HandlerFunc) error {
  aRoute, err := ws.FindRoute(url)
  if err != nil { return err }
  aRoute.DeleteHandler = handlerFunc
  return nil
}

type ESCFSByte func(bool, string) ([]byte, error)

func (ws *WS) ServeESCFSFile(
  w          http.ResponseWriter,
  req       *http.Request,
  filePath   string,
  fsByte     ESCFSByte,
) {
  useLocal := false
  if ! strings.HasPrefix(filePath, "/browserApp/") { useLocal = true }
  contentBytes, err := fsByte(useLocal, filePath)
  if err != nil {
    w.WriteHeader(404)
    ws.Log.Logf("could not find file [%s]", filePath)
    filePath = "/browserApp/static/404.html"
    contentBytes, err = fsByte(false, filePath)
    if err != nil {
      contentBytes = []byte(`
<html>
<head><title>Oops! That was not found!</title></head>
<body><h1>Oops! That was not found!</h1></body>
</html>
`)
    }
  }
  contentReader := bytes.NewReader(contentBytes)
  http.ServeContent(w, req, filePath, ws.InitTime, contentReader)
}


// Add a Get Handler for serving static files.
//
// This method adds the staticRoute and "/favicon.ico" (hidden) routes. 
//
// The "/favicon.ico" is served from the faviconPath, all other files are 
// served from the staticPath directory (which, if relative, is served 
// releative to the current directory). 
//
func (ws *WS) AddStaticFileHandlers(
  baseHtmlPath string,
  faviconPath  string,
  staticRoute  string, 
  staticPath   string,
  fsByte       ESCFSByte,
) error {

//TODO
// Use FSMustByte, bytes.NewReader, and http.ServeContent
//TODO

  ws.BaseRoute.GetHandler = func (w http.ResponseWriter, r *http.Request) {

    //
    // determine if we are replying in JSON
    //
    replyInJson := false
    for _, anAcceptValue := range r.Header["Accept"] {
      if strings.Contains(strings.ToLower(anAcceptValue), "json") {
        ws.Log.Logf("Replying in JSON [%s]\n\n", strings.Join(r.Header["Accept"], "|"))
        replyInJson = true
        break
      }
    }
    
    if replyInJson {
      ws.ReplyInJson(w, r, CreateRouteDesc(ws.BaseRoute))
      return 
    }
    
    ws.Log.Logf("serving [%s] as \"/\"", baseHtmlPath)
    ws.ServeESCFSFile(w, r, baseHtmlPath, fsByte)
  }
  
  err := ws.DescribeRoute("/favicon.ico", "The FavIcon", false)
  if err != nil {
    fmt.Errorf("Could not describe the route for /favicon.ico %w", err)
  } 

  err = ws.AddGetHandler(
    "/favicon.ico",
    func(w http.ResponseWriter, r *http.Request) {
      ws.Log.Logf("serving [%s] as favicon.ico", faviconPath)
      ws.ServeESCFSFile(w, r, faviconPath, fsByte)
    },
  )
  if err != nil {
    fmt.Errorf("Could not add getHandler for /favicon.ico %w", err)
  }
  
  err = ws.DescribeRoute(
    staticRoute,
    "Static resources such as the MithrilJS App",
    false,
  )
  if err != nil {
    fmt.Errorf("Could not describe the route for [%s] %w", staticRoute, err)
  }
  
  err = ws.AddGetHandler(
    staticRoute,
    func(w http.ResponseWriter, r *http.Request) {
      httpPath := r.URL.Path
      filePath := staticPath+strings.TrimPrefix(httpPath, staticRoute)
      ws.Log.Logf("serving [%s] from [%s]", httpPath, filePath)
      ws.ServeESCFSFile(w, r, filePath, fsByte)
    },
  )
  if err != nil {
    fmt.Errorf("Could not add getHandler for /static %w", err)
  }
  return nil
}

// Run the webserver at https://<host>:<port> using the TLS as
// configured in tlsConfig.
//
// NOTE: all routes must have been previously added using the DescribeRoute
// and AddxxxHandler methods.
//
func (ws *WS) RunWebServer() {
  ws.Server.Serve(ws.Listener)
}

// Find a route for the current URL Path using the FindRoute function, and
// then use the route's xxxHandler associated with the request method.
//
func (ws *WS) ServeHTTP(w http.ResponseWriter, r *http.Request) {
  ws.Log.Logf("url: [%s][%s] method: [%s]", r.URL.Path, r.URL.RawQuery, r.Method)

  aRoute, _ := ws.FindRoute(r.URL.Path)

  if aRoute == nil {
    http.Error(
      w,
      fmt.Sprintf("No route found for [%s] using [%s]",
        r.URL.Path,
        r.Method,
      ),
      http.StatusNotFound,
    )
    return
  }
  ws.Log.Logf(
    "Found route [%s](%s) for path [%s]", aRoute.Path, aRoute.Prefix, r.URL.Path,
  )
  
  method := r.Method
  query  := r.URL.Query()
  queryMethod := strings.Join(query["method"], "")
  if queryMethod != "" { method = strings.ToUpper(queryMethod) }

  switch method {
    case http.MethodGet    : if aRoute.GetHandler != nil {
      aRoute.GetHandler(w, r) ; return
    }
    case http.MethodHead   : if aRoute.HeadHandler != nil {
      aRoute.HeadHandler(w, r) ; return
    }
    case http.MethodPost   : if aRoute.PostHandler != nil {
      aRoute.PostHandler(w, r) ; return
    }
    case http.MethodPut    : if aRoute.PutHandler != nil {
      aRoute.PutHandler(w, r) ; return
    }
    case http.MethodDelete : if aRoute.DeleteHandler != nil {
      aRoute.DeleteHandler(w, r) ; return
    }
    default                :
      ws.Log.Logf("Incorrect RESTful HTTP Method [%s]", method)
      http.Error(
        w,
        fmt.Sprintf("Incorrect RESTful HTTP Method [%s]", method),
        http.StatusNotFound,
      )
     return
  }
  ws.Log.Logf(
    "No RESTful HTTP Handler found for [%s] using [%s]",
    r.URL.Path,
    r.Method,
  )
  http.Error(
    w,
    fmt.Sprintf("No RESTful HTTP Handler found for [%s] using [%s]",
      r.URL.Path,
      r.Method,
    ),
    http.StatusNotFound,
  )
}

////////////////////////////////////////////////////////////////////////
// Desribe route
//

