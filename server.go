package rest

import (
    "crypto/tls"
    "fmt"
    "github.com/maxmanuylov/utils/application"
    "net"
    "net/http"
    "strings"
    "time"
)

type Server struct {
    mux    *http.ServeMux
    prefix string
}

func NewServer() *Server {
    mux := http.NewServeMux()
    mux.Handle("/", http.NotFoundHandler())
    return &Server{mux: mux}
}

func (server *Server) CustomHandler(pattern string, handler http.Handler) {
    server.mux.Handle(server.path(pattern), handler)
}

func (server *Server) CustomHandlerFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
    server.mux.HandleFunc(server.path(pattern), handler)
}

func (server *Server) WithPrefix(prefix string) *Server {
    return &Server{
        mux: server.mux,
        prefix: server.path(fmt.Sprintf("/%s", strings.TrimPrefix(strings.TrimSuffix(prefix, "/"), "/"))),
    }
}

func (server *Server) path(path string) string {
    return fmt.Sprintf("%s%s", server.prefix, path)
}

func (server *Server) Listen(addr *net.TCPAddr) (net.Listener, error) {
    tcpListener, err := net.ListenTCP("tcp", addr)
    if err != nil {
        return nil, err
    }
    return tcpKeepAliveListener{tcpListener}, nil
}

func (server *Server) ListenTLS(addr *net.TCPAddr, config *tls.Config) (net.Listener, error) {
    innerListener, err := server.Listen(addr)
    if err != nil {
        return nil, err
    }
    return tls.NewListener(innerListener, config), nil
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
    defer listener.Close()

    go server.Serve(listener)

    application.WaitForTermination()

    return nil
}

func (server *Server) ListenAndServeTLS(addr *net.TCPAddr, config *tls.Config) error {
    listener, err := server.ListenTLS(addr, config)
    if err != nil {
        return err
    }
    defer listener.Close()

    go server.Serve(listener)

    application.WaitForTermination()

    return nil
}

func (server *Server) ListenAndServeFull(addr *net.TCPAddr, tlsAddr *net.TCPAddr, config *tls.Config) error {
    listener, err := server.Listen(addr)
    if err != nil {
        return err
    }
    defer listener.Close()

    tlsListener, err := server.ListenTLS(tlsAddr, config)
    if err != nil {
        return err
    }
    defer tlsListener.Close()

    go server.Serve(listener)
    go server.Serve(tlsListener)

    application.WaitForTermination()

    return nil
}
