package wiretap

import (
	"crypto/tls"
	"fmt"
	"github.com/fstab/h2c/cli/daemon"
	"github.com/fstab/h2c/http2client/frames"
	"github.com/fstab/http2/hpack"
	"io"
	"net"
	"os"
	"strings"
	"time"
)

// cert and key created with openssl req -x509 -newkey rsa:2048 -keyout key.pem -out cert.pem -nodes

const CERT = `-----BEGIN CERTIFICATE-----
MIIDsjCCApqgAwIBAgIJAPEIhIIN7mKdMA0GCSqGSIb3DQEBBQUAMEQxCzAJBgNV
BAYTAkRFMRMwEQYDVQQIEwpTb21lLVN0YXRlMQwwCgYDVQQKEwNoMmMxEjAQBgNV
BAMTCWxvY2FsaG9zdDAeFw0xNTA4MTQyMTA3NThaFw0xNTA5MTMyMTA3NThaMEQx
CzAJBgNVBAYTAkRFMRMwEQYDVQQIEwpTb21lLVN0YXRlMQwwCgYDVQQKEwNoMmMx
EjAQBgNVBAMTCWxvY2FsaG9zdDCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoC
ggEBALxe4gsFS7Q/swsA5wOmUKR0u9O+IrRnMvGbiKMA7qluMZ1Vf2VriNk7Vf3P
uftb5DaEHqT69dVbCQzgq3OoZutLFeC91lGwLrDzeMZK3oiJ9R8bWNA6FI7Oavvg
EHU4PW88bvlkzkqsUbnA1gl7vQoBofdEUS8fFg42vRcE+XOL4eWoN5vva9kX8ZNZ
vhrjQI9NPnJqumq7czGNQQ7bRiDePlVv6Ng6+sRcJTpCWnoX6p742scO5FhBcqQy
RzJmnDXvl+W726VuM7Lxz6W+N15uVCcuuIIcox6uoKX+3uLKZwf7G1P7MfVK1EAS
Uhg7dNpUXTkZaV7bQN+7QN8UsekCAwEAAaOBpjCBozAdBgNVHQ4EFgQUIyx/hAi/
o+lGfLwbO7roeysVrXEwdAYDVR0jBG0wa4AUIyx/hAi/o+lGfLwbO7roeysVrXGh
SKRGMEQxCzAJBgNVBAYTAkRFMRMwEQYDVQQIEwpTb21lLVN0YXRlMQwwCgYDVQQK
EwNoMmMxEjAQBgNVBAMTCWxvY2FsaG9zdIIJAPEIhIIN7mKdMAwGA1UdEwQFMAMB
Af8wDQYJKoZIhvcNAQEFBQADggEBAApp8GgCQgBEArUD8pTQiG6L43zMMnDYER8I
aFLeD3x/2N1+ZNXStIRZcGEqd/CsQqF2nl0thA7h8sdG97OMokhum6SJzITVRW7O
HeRTn3/laU57AuaV4g/WMWghvYBuH+LQUHkg5rwJ9uij7LU4cfvEZHFKHTHXVTh4
d26jyJ3Sm8hfoSGU3vJ/KSwYmjZGHbyev4JtXYjoen53CqXYyCnhcyBWhdxCbPg/
OsUtIMz0L98UIkxB8bn2CEKTJZ+crWdCKmLm2ZBkOokHy9Cu8fNcLQapzIupUVE8
WKGxhKUd1Sx20Ou5t0Z1vWIISsaHvWuA7Dt/7YJoCjv1de7BNbM=
-----END CERTIFICATE-----`

const KEY = `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAvF7iCwVLtD+zCwDnA6ZQpHS7074itGcy8ZuIowDuqW4xnVV/
ZWuI2TtV/c+5+1vkNoQepPr11VsJDOCrc6hm60sV4L3WUbAusPN4xkreiIn1HxtY
0DoUjs5q++AQdTg9bzxu+WTOSqxRucDWCXu9CgGh90RRLx8WDja9FwT5c4vh5ag3
m+9r2Rfxk1m+GuNAj00+cmq6artzMY1BDttGIN4+VW/o2Dr6xFwlOkJaehfqnvja
xw7kWEFypDJHMmacNe+X5bvbpW4zsvHPpb43Xm5UJy64ghyjHq6gpf7e4spnB/sb
U/sx9UrUQBJSGDt02lRdORlpXttA37tA3xSx6QIDAQABAoIBAHiqGwhOOyFakJli
2ZjH5+6A7HSF3ntLhCGqrQslpLwZ32RWiUXxbXciAEKW1x8BzR1y4qJyNBmYuCJs
pMdwv4TH01rkoC4xuqcVP29eNFCodsGXZfv0yIh2B6gS7tf3z7q6rqfJADBrU+q2
cSUgA4cuZo8bTxntQtaWD+h4AkcV4LC3XqttUtH3mmMSrv38r/LlnmD0t1CxkE6E
N6ByOJOHK5ORG//H5kkOothNGPC2YOVw4WITE5RHWMP/8B9sHdQXuGxIHBcWWoxE
0J0odwpoZ6vF7+nOqrOOew2dSRuPI4eir1zQqVIwffzLz4kTP31JSbnbxVsZ1kAz
V+sCy3UCgYEA7oUXOeHNDKAkXtylgsJ2XYTbuU9FpeIjpUBPgYx5F3veobpHeG8J
KU757EJrYnIAtBtj9F1NX92F/yx5nk5uzUjlL61I8wuD5ncAyVY30DpWM7CaAaWl
dbUFIWAYPHK7AvVcqO4p2CpFE1s0YT3ekJ2qAvPnA7jVu2c/dAocbrsCgYEAyizv
LB2kM/F2SaAE4J8VtEcdc48x1eeWm/gBvKCDwGTvOc1F8aDNh7c5ZzQZuOA1NI0x
haZ+ygsPYKQatvsT9INBXw4hsXZB7qBDdtS/WdLg31RUsw6+/xpq4qOq8WOyrrnM
4IS1x+fOYGKmzPw58lOA+LayIve7ydEXB654AasCgYEAw26lYzXSTuAALQHZU1SG
q4WqiyGazZqG3mXdPyacKVPDTPxWhyVjekdNm/moBxel3+z5b0XrmfrmSfhlBgL5
4pYxw2jWdt4eiv1C1bUhMio6a0vuRB83fUR/GaOk+BKBjKEB9SB/hLDNvFhkiLCq
5g9pN9YkmPYfmde1NBz8wvUCgYAc8IOn1/JaMRUSguJP2NW9gXR4xyWGDelkGAL/
oiZZ0tjfeD+rz6274IFKAY4xBX74L8HH9MYvW5fu6G6ehKAdnvArkBVIlrnnU290
wg1F6UahESwymUjDsV9dY7ojZXb9RcFK3hQ7MjY7W8OukeglhMhwUY58LOPnhpN6
WQH6kwKBgQCs4QRrDdiVqmnxGcIfLdVCeX0Cq32RsL9Rlmaj6nL4zUR6bwq7TUGW
zPLXVYepsNYyJEPAJd7QHzod0dDESJT3E7hm5xULdZiIU2/HR9D7IY9ukvaOHBz7
zuaJapKwoKydbiLGlhvpcBtsE5ahRo3A62ckaeVi4JX7dXcpqAtM9A==
-----END RSA PRIVATE KEY-----`

const CLIENT_PREFACE = "PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n"

func Run(local string, remote string) error {
	if !strings.Contains(remote, ":") {
		remote = remote + ":443"
	}
	listener, err := net.Listen("tcp", local)
	if err != nil {
		return err
	}
	dumpIncoming := make(chan frames.Frame)
	dumpOutgoing := make(chan frames.Frame)
	go dumpFrames(dumpIncoming, dumpOutgoing)
	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}
		go func() {
			err := handleConnection(conn, local, remote, dumpIncoming, dumpOutgoing)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error while handling connection: %v\n", err.Error())
			}
		}()
	}
}

func handleConnection(conn net.Conn, local, remote string, dumpIncoming, dumpOutgoing chan frames.Frame) error {
	clientConn, err := negotiateH2Protocol(conn)
	if err != nil {
		return err
	}
	err = receiveClientPreface(clientConn)
	if err != nil {
		return err
	}
	serverConn, err := connectToServer(remote)
	if err != nil {
		return err
	}
	go forwardFrames(clientConn, serverConn, remote, dumpOutgoing)
	go forwardFrames(serverConn, clientConn, local, dumpIncoming)
	return nil
}

func dumpFrames(in chan frames.Frame, out chan frames.Frame) {
	for {
		select {
		case frame := <-in:
			daemon.DumpIncoming(frame)
		case frame := <-out:
			daemon.DumpOutgoing(frame)
		}
	}
}

func forwardFrames(from net.Conn, to net.Conn, remoteAuthority string, dump chan frames.Frame) {
	defer from.Close()
	defer to.Close()
	encodingContext := frames.NewEncodingContext()
	decodingContext := frames.NewDecodingContext()
	for {
		frame, err := readNextFrame(from, decodingContext)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error while reading next frame: %v\n", err.Error())
			fmt.Fprintf(os.Stderr, "Closing connection.\n")
			return
		}
		fixAuthorityHeader(frame, remoteAuthority)
		dump <- frame
		err = writeFrame(to, frame, encodingContext)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error while forwarding next frame: %v\n", err.Error())
			fmt.Fprintf(os.Stderr, "Closing connection.\n")
			return
		}
	}
}

func getHeaders(headersFrame *frames.HeadersFrame, pushPromiseFrame *frames.PushPromiseFrame) []hpack.HeaderField {
	if headersFrame != nil {
		return headersFrame.Headers
	} else {
		return pushPromiseFrame.Headers
	}
}

func setHeaders(headersFrame *frames.HeadersFrame, pushPromiseFrame *frames.PushPromiseFrame, headers []hpack.HeaderField) {
	if headersFrame != nil {
		headersFrame.Headers = headers
	} else {
		pushPromiseFrame.Headers = headers
	}
}

// The web browser will access https://localhost:8443, so it will set the :authority header to localhost.
// We need to replace this with the remote host to get a valid request for the remote host.
func fixAuthorityHeader(frame frames.Frame, remoteAuthority string) {
	headersFrame, isHeaders := frame.(*frames.HeadersFrame)
	pushPromiseFrame, isPushPromise := frame.(*frames.PushPromiseFrame)
	if isHeaders || isPushPromise {
		origHeaders := getHeaders(headersFrame, pushPromiseFrame)
		fixedHeaders := make([]hpack.HeaderField, len(origHeaders))
		for i, header := range origHeaders {
			if header.Name == ":authority" {
				header = hpack.HeaderField{
					Name:  ":authority",
					Value: remoteAuthority,
				}
			}
			fixedHeaders[i] = header
		}
		setHeaders(headersFrame, pushPromiseFrame, fixedHeaders)
	}
}

// TODO: copy-and-paste from connection
func readNextFrame(conn net.Conn, context *frames.DecodingContext) (frames.Frame, error) {
	headerData := make([]byte, 9) // Frame starts with a 9 Bytes header
	_, err := io.ReadFull(conn, headerData)
	if err != nil {
		return nil, err
	}
	header := frames.DecodeHeader(headerData)
	payload := make([]byte, header.Length)
	_, err = io.ReadFull(conn, payload)
	if err != nil {
		return nil, err
	}
	decodeFunc := frames.FindDecoder(frames.Type(header.HeaderType))
	if decodeFunc == nil {
		return nil, fmt.Errorf("%v: Unknown frame type.", header.HeaderType)
	}
	return decodeFunc(header.Flags, header.StreamId, payload, context)
}

// TODO: copy-and-paste from connection
func writeFrame(conn net.Conn, frame frames.Frame, context *frames.EncodingContext) error {
	encodedFrame, err := frame.Encode(context)
	if err != nil {
		return fmt.Errorf("Failed to write frame: %v", err.Error())
	}
	_, err = conn.Write(encodedFrame)
	if err != nil {
		return fmt.Errorf("Failed to write frame: %v", err.Error())
	}
	return nil
}

func negotiateH2Protocol(conn net.Conn) (*tls.Conn, error) {
	keyPair, err := tls.X509KeyPair([]byte(CERT), []byte(KEY))
	if err != nil {
		return nil, err
	}
	tlsConn := tls.Server(conn, &tls.Config{
		Certificates: []tls.Certificate{keyPair},
		NextProtos:   []string{"h2"},
	})
	err = tlsConn.Handshake()
	if err != nil {
		return nil, err
	}
	state := tlsConn.ConnectionState()
	if state.NegotiatedProtocol != "h2" {
		return nil, fmt.Errorf("Client does not support HTTP/2 protocol.")
	}
	return tlsConn, nil
}

func receiveClientPreface(tlsConn *tls.Conn) error {
	buf := make([]byte, len(CLIENT_PREFACE))
	_, err := io.ReadFull(tlsConn, buf)
	if err != nil {
		return fmt.Errorf("Failed to receive data from %v: %v", tlsConn.RemoteAddr(), err.Error())
	}
	if string(buf) != CLIENT_PREFACE {
		firstLine := strings.Split(string(buf), "\n")[0]
		return fmt.Errorf("Failed to receive data from %v: Received \"%v\".", tlsConn.RemoteAddr(), firstLine)
	}
	return nil
}

func connectToServer(hostAndPort string) (*tls.Conn, error) {
	dialerWithTimeout := &net.Dialer{Timeout: 5 * time.Second}
	tlsConn, err := tls.DialWithDialer(dialerWithTimeout, "tcp", hostAndPort, &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"h2"},
	})
	if err != nil {
		return nil, fmt.Errorf("Failed to connect to %v: %v", hostAndPort, err.Error())
	}
	if tlsConn.ConnectionState().NegotiatedProtocol != "h2" {
		return nil, fmt.Errorf("Server does not support HTTP/2 protocol.")
	}
	_, err = tlsConn.Write([]byte(CLIENT_PREFACE))
	if err != nil {
		return nil, fmt.Errorf("Failed to write client preface to %v: %v", hostAndPort, err.Error())
	}
	return tlsConn, nil
}
