package rest

import (
    "github.com/maxmanuylov/utils/application"
    "net"
    "net/http"
    "time"
)

type Server struct {
    mux *http.ServeMux
}

func NewServer() *Server {
    mux := http.NewServeMux()
    mux.Handle("/", http.NotFoundHandler())
    return &Server{mux: mux}
}

func (server *Server) CustomHandler(pattern string, handler http.Handler) {
    server.mux.Handle(pattern, handler)
}

func (server *Server) CustomHandlerFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
    server.mux.HandleFunc(pattern, handler)
}

func (server *Server) Listen(addr *net.TCPAddr) (net.Listener, error) {
    tcpListener, err := net.ListenTCP("tcp", addr)
    if err != nil {
        return nil, err
    }
    return tcpKeepAliveListener{tcpListener}, nil
}

func (server *Server) Serve(listener net.Listener) {
    httpServer := &http.Server{Handler: server.mux}
    httpServer.Serve(listener)
}

type tcpKeepAliveListener struct {
    *net.TCPListener
}

func (listener tcpKeepAliveListener) Accept() (net.Conn, error) {
    tcpConnection, err := listener.AcceptTCP()
    if err != nil {
        return nil, err
    }

    tcpConnection.SetKeepAlive(true)
    tcpConnection.SetKeepAlivePeriod(3 * time.Minute)

    return tcpConnection, nil
}

func (server *Server) ListenAndServe(addr *net.TCPAddr) error {
    listener, err := server.Listen(addr)
    if err != nil {
        return err
    }

    go server.Serve(listener)

    application.WaitForTermination()

    listener.Close()

    return nil
}
