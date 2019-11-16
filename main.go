package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"strings"

	bn256 "github.com/ethereum/go-ethereum/crypto/bn256/cloudflare"
)

var (
	help   bool
	input  string
	output string
)

func init() {
	flag.BoolVar(&help, "h", false, "this help")
	flag.StringVar(&input, "i", "input.json", "input json file of delta, a, b and c")
	flag.StringVar(&output, "o", "output.json", "output json file of b and c")
	flag.Usage = usage
}

func usage() {
	fmt.Fprintf(os.Stderr, `gen-g16-proof version: gen-g16-proof/1.0.0
Usage: gen-g16-proof -i input.json -o output.json

Options:
`)
	flag.PrintDefaults()
}

func parseBigInt(s string) *big.Int {
	err := fmt.Errorf("failed to parse %s", s)
	var base int
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		s = s[2:]
		base = 16
	} else if strings.HasPrefix(s, "0") {
		s = s[1:]
		base = 8
	} else if strings.HasPrefix(s, "b") || strings.HasPrefix(s, "B") {
		s = s[1:]
		base = 2
	} else {
		base = 10
	}
	if ret, ok := new(big.Int).SetString(s, base); !ok {
		panic(err)
	} else {
		return ret
	}
}

func parseG2Point(s [2][2]string) *bn256.G2 {
	xIm := parseBigInt(s[0][0])
	xRe := parseBigInt(s[0][1])
	yIm := parseBigInt(s[1][0])
	yRe := parseBigInt(s[1][1])
	p := new(bn256.G2)
	b := make([]byte, 32*4)

	xbIm := xIm.Bytes()
	xbRe := xRe.Bytes()
	ybIm := yIm.Bytes()
	ybRe := yRe.Bytes()

	copy(b[1*32-len(xbIm):1*32], xbIm)
	copy(b[2*32-len(xbRe):2*32], xbRe)
	copy(b[3*32-len(ybIm):3*32], ybIm)
	copy(b[4*32-len(ybRe):4*32], ybRe)
	_, err := p.Unmarshal(b)
	if err != nil {
		panic(err)
	}
	return p
}

func hexG2Point(p *bn256.G2) [2][2]string {
	b := p.Marshal()
	xIm := b[0*32 : 1*32]
	xRe := b[1*32 : 2*32]
	yIm := b[2*32 : 3*32]
	yRe := b[3*32 : 4*32]
	return [2][2]string{
		[2]string{
			"0x" + hex.EncodeToString(new(big.Int).SetBytes(xIm).Bytes()),
			"0x" + hex.EncodeToString(new(big.Int).SetBytes(xRe).Bytes()),
		},
		[2]string{
			"0x" + hex.EncodeToString(new(big.Int).SetBytes(yIm).Bytes()),
			"0x" + hex.EncodeToString(new(big.Int).SetBytes(yRe).Bytes()),
		},
	}
}

func parseG1Point(s [2]string) *bn256.G1 {
	x := parseBigInt(s[0])
	y := parseBigInt(s[1])
	p := new(bn256.G1)
	b := make([]byte, 32*2)
	xb := x.Bytes()
	yb := y.Bytes()
	copy(b[1*32-len(xb):1*32], xb)
	copy(b[2*32-len(yb):2*32], yb)
	_, err := p.Unmarshal(b)
	if err != nil {
		panic(err)
	}
	return p
}

func hexG1Point(p *bn256.G1) [2]string {
	b := p.Marshal()
	x := b[0*32 : 1*32]
	y := b[1*32 : 2*32]
	return [2]string{
		"0x" + hex.EncodeToString(new(big.Int).SetBytes(x).Bytes()),
		"0x" + hex.EncodeToString(new(big.Int).SetBytes(y).Bytes()),
	}
}

type InputData struct {
	Delta [2][2]string `json:"delta"`
	A     [2]string    `json:"a"`
	B     [2][2]string `json:"b"`
	C     [2]string    `json:"c"`
	Eta   string       `json:"eta"`
}

type OutputData struct {
	B   [2][2]string `json:"b"`
	C   [2]string    `json:"c"`
	Eta string       `json:"eta"`
}

func main() {
	flag.Parse()
	if help {
		flag.Usage()
		return
	}
	data, err := ioutil.ReadFile(input)
	if err != nil {
		panic(err)
	}
	inputData := new(InputData)
	err = json.Unmarshal(data, inputData)
	if err != nil {
		panic(err)
	}

	Delta := parseG2Point(inputData.Delta)
	B := parseG2Point(inputData.B)
	A := parseG1Point(inputData.A)
	C := parseG1Point(inputData.C)
	Eta := new(big.Int)
	if inputData.Eta == "" {
		Eta, _, err = bn256.RandomG1(rand.Reader)
		if err != nil {
			panic(err)
		}
	} else {
		Eta = parseBigInt(inputData.Eta)
	}

	outputData := new(OutputData)

	outputData.B = hexG2Point(new(bn256.G2).Add(Delta, new(bn256.G2).ScalarMult(B, Eta)))
	outputData.C = hexG1Point(new(bn256.G1).Add(C, new(bn256.G1).ScalarMult(A, Eta)))
	outputData.Eta = "0x" + hex.EncodeToString(Eta.Bytes())

	data, err = json.Marshal(outputData)
	if err != nil {
		panic(err)
	}
	err = ioutil.WriteFile(output, data, 0644)
	if err != nil {
		panic(err)
	}
}
