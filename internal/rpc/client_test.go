package rpc

import (
	. "cess-bucket/internal/rpc/proto"
	"context"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"
)

type testService struct{}

func (testService) HelloAction(body []byte) (proto.Message, error) {
	fmt.Println(string(body))
	return &RespBody{Code: 0, Msg: "hi, i am server!"}, nil
}

func TestDialWebsocket(t *testing.T) {
	srv := NewServer()
	srv.Register("test", testService{})
	s := httptest.NewServer(srv.WebsocketHandler([]string{"*"}))
	defer s.Close()
	defer srv.Close()

	wsURL := "ws:" + strings.TrimPrefix(s.URL, "http:")
	fmt.Println(wsURL)
	client, err := DialWebsocket(context.Background(), wsURL, "")
	if err != nil {
		t.Fatal(err)
	}

	req := &ReqMsg{
		Service: "test",
		Method:  "hello",
		Body:    []byte("hi, i am client!"),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	resp, err := client.Call(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()
	var body RespBody
	proto.Unmarshal(resp.Body, &body)
	fmt.Println(body.Msg)
}
