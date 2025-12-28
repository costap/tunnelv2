package client

import (
	"bytes"
	"github.com/chzyer/test"
	"go.uber.org/zap"
	"golang.org/x/net/nettest"
	"io"
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestConnectionHandler_RunSimple(t *testing.T) {
	ln, err := nettest.NewLocalListener("tcp")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close()

		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		if err != nil {
			t.Fatal(err)
		}
		if string(buf[:n]) != "ping" {
			t.Fatal("expected ping")
		}
		_, err = conn.Write([]byte("pong"))
		if err != nil {
			t.Fatal(err)
		}
	}()

	in, out := make(chan []byte), make(chan []byte)
	h := NewConnectionHandler(zap.NewNop(), "aconnId", ln.Addr().String(), in, out)
	//defer h.Close()
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		h.Run()
		wg.Done()
	}()

	in <- []byte("ping")

	data := <-out
	h.Close()

	if string(data) != "pong" {
		t.Fatalf("expected %s, got %s", "pong", data)
	}

	wg.Wait()
	if runtime.NumGoroutine() > 2 {
		// create a new buffer
		buf := make([]byte, 1<<20)
		i := runtime.Stack(buf, true)
		t.Logf("goroutines: %s", buf[:i])
		t.Fatal("expected running to be 2, got", runtime.NumGoroutine())
	}
}

// write a fuzzing test to test the connection handler
func FuzzConnectionHandler_Run(f *testing.F) {
	l, _ := zap.NewDevelopment()
	//        l := zap.NewNop()
	test.RandBytes(1024)
	tt := [][][]byte{
		//{[]byte("ping"), []byte("pong")},
		//{{1, 2, 3, 4, 5, 6, 7, 8, 9, 0}, {1, 2, 3, 4, 5, 6, 7, 8, 9, 0}},
		//{test.RandBytes(1024), test.RandBytes(1024)},
		{[]byte("\x01\x02\x03\x04\x05\x06\a\b\t\x00"), []byte("")},
	}
	for _, bytes := range tt {
		f.Add(bytes[0], bytes[1])
	}
	f.Fuzz(func(t *testing.T, inBytes, outBytes []byte) {
		ln, err := nettest.NewLocalListener("tcp")
		if err != nil {
			t.Fatal(err)
		}
		defer ln.Close()

		var readBytes []byte
		go func() {
			conn, err := ln.Accept()
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()

			buf := make([]byte, 1024)
			conn.SetReadDeadline(time.Now().Add(5 * time.Second))
			for len(readBytes) < len(inBytes) {
				n, err := conn.Read(buf)
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatal(err)
				}
				readBytes = append(readBytes, buf[:n]...)
			}

			_, err = conn.Write(outBytes)
			if err != nil {
				t.Fatal(err)
			}
		}()

		in, out := make(chan []byte), make(chan []byte)
		h := NewConnectionHandler(l, "aconnId", ln.Addr().String(), in, out)
		//defer h.Close()
		wg := &sync.WaitGroup{}
		wg.Add(1)
		go func() {
			h.Run()
			wg.Done()
		}()

		in <- inBytes

		data := <-out
		h.Close()
		wg.Wait()

		if bytes.Compare(data, outBytes) != 0 {
			t.Fatal("expected 0, got", bytes.Compare(data, outBytes))
		}

		if bytes.Compare(readBytes, inBytes) != 0 {
			t.Fatalf("expected %s, got %s", string(readBytes), string(inBytes))
		}

		//if runtime.NumGoroutine() > 5 {
		//	// create a new buffer
		//	buf := make([]byte, 1<<20)
		//	i := runtime.Stack(buf, true)
		//	t.Logf("goroutines: %s", buf[:i])
		//	t.Fatal("expected running to be 3, got", runtime.NumGoroutine())
		//}
	})
}
