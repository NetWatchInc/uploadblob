package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image/png"
	"io"
	"log"
	"net/http"
	"os"

	atp "github.com/bluesky-social/indigo/api/atproto"
	lexutil "github.com/bluesky-social/indigo/lex/util"
	"github.com/bluesky-social/indigo/util/cliutil"
	xrpc "github.com/bluesky-social/indigo/xrpc"
	"github.com/urfave/cli/v2"
)

var cctx *cli.Context
var client *xrpc.Client

func main() {
	// Enter the filename from the local directory
	fileName := ""

	client, err := cliutil.GetXrpcClient(cctx, false)
	if err != nil {
		log.Fatal("Error getting XRPC client:", err)
	}
	client.Host = "https://bsky.social"

	xrpcc, err := authenticateSession(client)
	if err != nil {
		log.Fatal("Error authenticating:", err)
	}

	// Read the file
	file, err := os.Open(fileName)
	if err != nil {
		log.Fatalf("failed to read file: %v", err)
	}

	image, err := png.Decode(file)
	if err != nil {
		log.Fatal("Error decoding PNG: ", err)
		file.Close()
	}
	file.Close()

	var imageBuffer bytes.Buffer
	err = png.Encode(&imageBuffer, image)
	if err != nil {
		log.Fatal("Error encoding buffer: ", err)
	}
	uploadBlob(imageBuffer, xrpcc)
}

func uploadBlob(png bytes.Buffer, xrpcc *xrpc.Client) (*lexutil.LexBlob, error) {
	var blob lexutil.LexBlob
	req, err := http.NewRequest("POST", "https://bsky.social/xrpc/com.atproto.repo.uploadBlob", &png)
	if err != nil {
		log.Fatalf("Error creating request: %v", err)
	}
	req.Header.Set("Content-Type", "image/png")
	req.Header.Set("Authorization", "Bearer "+xrpcc.Auth.AccessJwt)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error POSTing blob: %v\n, Response: %v\n", err, resp)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Server responded with status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	fmt.Println(string(body))

	if err != nil {
		log.Fatalf("Error reading response body: %v", err)
	}

	var objmap map[string]json.RawMessage
	if err := json.Unmarshal(body, &objmap); err != nil {
		log.Fatalf("Error with basic json parsing: %v", err)
	}

	if err := blob.UnmarshalJSON(objmap["blob"]); err != nil {
		log.Fatalf("Error parsing response body: %v", err)
	}

	return &blob, nil
}

func authenticateSession(xrpcc *xrpc.Client) (*xrpc.Client, error) {
	ses, err := atp.ServerCreateSession(context.TODO(), xrpcc, &atp.ServerCreateSession_Input{
		// Enter Bluesky handle & app password
		Identifier: "",
		Password:   "",
	})
	xrpcc.Auth = &xrpc.AuthInfo{
		AccessJwt:  ses.AccessJwt,
		RefreshJwt: ses.RefreshJwt,
		Handle:     ses.Handle,
		Did:        ses.Did,
	}
	if err != nil {
		log.Fatal("Error creating session: ", err)
	}
	return xrpcc, nil
}
