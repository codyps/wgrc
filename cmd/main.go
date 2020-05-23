//go:generate protoc --go_out=plugins=grpc:api stream.proto
package main
import (
	"log"
	"net"

	"github.com/jmesmon/wgrc/api"
)

func main() {
	fmt.Println("hi")
}
