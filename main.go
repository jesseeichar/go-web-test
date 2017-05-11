package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

var nics []string = getNicAddresses()

type Response struct {
	Port int
	Protocol, Host,
	RemoteAddr, RequestBody string
	Nics []string
}

func createResponse(protocol string, port int, r *http.Request) []byte {
	hostname, err := os.Hostname()
	checkError(protocol+": get Hostname", err)
	body, err := ioutil.ReadAll(r.Body)
	checkError(protocol+": get request body", err)

	logRequest(protocol, r.RemoteAddr, string(body))
	response := Response{
		Port:        port,
		Protocol:    r.Proto,
		Host:        hostname,
		RemoteAddr:  r.RemoteAddr,
		RequestBody: string(body),
		Nics:        nics,
	}

	jsonBytes, err := json.MarshalIndent(response, "", "    ")
	checkError(protocol+": marshal http JSON response", err)

	return jsonBytes
}

func main() {
	httpPort := flag.Int("httpPort", 80, "http port")
	httpsPort := flag.Int("httpsPort", 443, "https port")
	udpPort := flag.Int("udpPort", 9090, "udp port")
	serverCertFile := flag.String("cert", "server.crt", "location of server certificate to use")
	serverKeyFile := flag.String("key", "server.key", "location of private key to use")

	flag.Parse()

	writeCerts(*serverCertFile, *serverKeyFile)

	go startHttp(*httpPort)
	go startHttps(*httpsPort)
	startUDPListener(*udpPort)
}

func createHTTPHandler(port int, protocol string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		responseJSON := createResponse(protocol, port, r)
		if strings.Contains(r.Header.Get("Accept"), "html") {
			w.Header().Set("Content-Type", "text/html")
			responseHTML := fmt.Sprintf("<html><body><pre>%s</pre></body></html>", string(responseJSON))
			w.Write([]byte(responseHTML))
		} else {
			w.Header().Add("Content-Type", "application/json")
			w.Write(responseJSON)
		}
	})
}

func startUDPListener(port int) {
	fmt.Printf("Starting udp on port %v\n", port)
	serverAddr, err := net.ResolveUDPAddr("udp", address(port))
	checkError("Resolve udp address", err)

	/* Now listen at selected port */
	serverConn, err := net.ListenUDP("udp", serverAddr)
	checkError("Start udp listener", err)
	defer serverConn.Close()

	buf := make([]byte, 1024)

	for {
		n, addr, err := serverConn.ReadFromUDP(buf)
		checkError("Reading data from udp", err)

		body := string(buf[0:n])
		logRequest("UDP", addr.String(), body)

		writeUDPResponse(addr.String(), body, port)
	}
}

func writeUDPResponse(addr, body string, port int) {
	conn, err := net.Dial("udp", addr)
	if err != nil {
		checkError("Write UDP response", err)
	}
	defer conn.Close()
	hostname, err := os.Hostname()
	checkError("UDP: get Hostname", err)

	response := Response{
		Port:        port,
		Protocol:    "UDP",
		Host:        hostname,
		RemoteAddr:  addr,
		RequestBody: string(body),
		Nics:        nics,
	}

	jsonBytes, err := json.MarshalIndent(response, "", "    ")
	checkError("UDP: marshal http JSON response", err)

	conn.SetDeadline(time.Now().Add(3 * time.Second))
	conn.Write(jsonBytes)
}

func startHttp(port int) {
	fmt.Printf("Starting http on port %v\n", port)
	err := http.ListenAndServe(address(port), createHTTPHandler(port, "http"))
	checkError("Start http listening", err)
}
func startHttps(port int) {
	fmt.Printf("Starting https on port %v\n", port)
	err := http.ListenAndServeTLS(address(port), "server.crt", "server.key", createHTTPHandler(port, "https"))
	checkError("Start https listening", err)
}

func checkError(desc string, err error) {
	if err != nil {
		fmt.Println("Error: "+desc, err)
		os.Exit(0)
	}
}

func logRequest(protocol, address, body string) {
	fmt.Printf("%s - %s, Message from %s recieved data: %s\n\n", strings.ToUpper(protocol), time.Now().Format(time.UnixDate), address, body)

}

func address(port int) string {
	return fmt.Sprintf(":%v", port)
}

func getNicAddresses() []string {
	ifaces, err := net.Interfaces()
	nics := make([]string, 1)
	if err != nil {
		return nics
	}

	// handle err
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err == nil {
			for _, addr := range addrs {
				switch v := addr.(type) {
				case *net.IPNet:
					nics = append(nics, v.IP.String())
				case *net.IPAddr:
					nics = append(nics, v.IP.String())
				}
			}
		}
	}

	return nics
}

func writeCerts(crt, key string) {
	if _, err := os.Stat(crt); os.IsNotExist(err) {
		err := ioutil.WriteFile(crt, []byte(serverCert), os.FileMode(0600))
		checkError("Error writing "+crt, err)
	}
	if _, err := os.Stat(key); os.IsNotExist(err) {
		err := ioutil.WriteFile(key, []byte(serverKey), os.FileMode(0600))
		checkError("Error writing "+key, err)
	}
}

var serverCert = `-----BEGIN CERTIFICATE-----
MIICbTCCAfKgAwIBAgIJAJv2Mek1scV1MAoGCCqGSM49BAMCMHQxCzAJBgNVBAYT
AkNIMQ0wCwYDVQQIDARCZXJuMQ0wCwYDVQQHDARCZXJuMRAwDgYDVQQKDAdNaW1h
Y29tMQwwCgYDVQQLDANEZXYxJzAlBgkqhkiG9w0BCQEWGGplc3NlLmVpY2hhckBt
aW1hY29tLmNvbTAeFw0xNzA1MDUwOTMzMDBaFw0yNzA1MDMwOTMzMDBaMHQxCzAJ
BgNVBAYTAkNIMQ0wCwYDVQQIDARCZXJuMQ0wCwYDVQQHDARCZXJuMRAwDgYDVQQK
DAdNaW1hY29tMQwwCgYDVQQLDANEZXYxJzAlBgkqhkiG9w0BCQEWGGplc3NlLmVp
Y2hhckBtaW1hY29tLmNvbTB2MBAGByqGSM49AgEGBSuBBAAiA2IABEY+tJxMgzlR
5jNci0RuXyt3HB8aSzHZopYEyus01uphVN1MqNUbNxCSpmk/xzWBOD8VhoAFHEuf
cHHAXmSQD81fCM1MnKbC1rgB0PFR1OznlG03EutOqQlj4BbD84P+qKNQME4wHQYD
VR0OBBYEFAzToagaBKUEOGV1UPG8T4FS8EpqMB8GA1UdIwQYMBaAFAzToagaBKUE
OGV1UPG8T4FS8EpqMAwGA1UdEwQFMAMBAf8wCgYIKoZIzj0EAwIDaQAwZgIxAJIc
hrNwCJXnxgIzLk92Xu5c89Vhb9Fr4w0OzJ+mLUIjOxldkVu7Cuw6RsX61CAqfwIx
AN+E0piWsvlh2R9OlCg7f6Uns/gNTTN+XYdbBuZ1JVTAEqb3RKwLiQ1/23JH6UxA
EA==
-----END CERTIFICATE-----`

var serverKey = `-----BEGIN EC PARAMETERS-----
BgUrgQQAIg==
-----END EC PARAMETERS-----
-----BEGIN EC PRIVATE KEY-----
MIGkAgEBBDBcimIa0LWzy7GA7vBtIGkTuFCFe6Nhj6RE1cEzhNoxQ862uhMDRuTP
b5DpARkCsrGgBwYFK4EEACKhZANiAARGPrScTIM5UeYzXItEbl8rdxwfGksx2aKW
BMrrNNbqYVTdTKjVGzcQkqZpP8c1gTg/FYaABRxLn3BxwF5kkA/NXwjNTJymwta4
AdDxUdTs55RtNxLrTqkJY+AWw/OD/qg=
-----END EC PRIVATE KEY-----`
