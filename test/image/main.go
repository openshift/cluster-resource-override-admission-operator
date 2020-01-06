package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

const (
	ImageFormatVar       = "IMAGE_FORMAT"
	ComponentPlaceholder = "${component}"

	OperatorName = "cluster-resource-override-admission-operator"
	OperandName  = "cluster-resource-override-admission"
)

var (
	mode   = flag.String("mode", "", "operation mode, either local or ci")
	output = flag.String("output", "", "path to file where image URLs are written to")
)

type Image struct {
	Operator string `json:"operator,omitempty"`
	Operand  string `json:"operand,omitempty"`
}

type ImageURLGetter func() (image *Image, err error)

func main() {
	flag.Parse()

	if *output == "" {
		log.Fatal("output file name can not be empty")
	}

	work := func(getter ImageURLGetter) error {
		image, err := getter()
		if err != nil {
			return err
		}

		return write(image, *output)
	}

	var err error
	switch *mode {
	case "local":
		err = work(getLocalImageURL)
	case "ci":
		err = work(getCIImageURL)
	default:
		log.Fatalf("unsupported mode, value=%s", *mode)
	}

	if err != nil {
		log.Fatalf("ran in to error - %s", err.Error())
	}

	log.Print("successfully wrote image URL(s)")
	os.Exit(0)
}

func getCIImageURL() (image *Image, err error) {
	format := os.Getenv(ImageFormatVar)
	if format == "" {
		err = fmt.Errorf("ENV var %s not defined", ImageFormatVar)
		return
	}

	image = &Image{
		Operator: strings.ReplaceAll(format, ComponentPlaceholder, OperatorName),
		Operand:  strings.ReplaceAll(format, ComponentPlaceholder, OperandName),
	}

	return
}

func getLocalImageURL() (image *Image, err error) {
	const (
		LocalOperatorImageVar = "CLUSTERRESOURCEOVERRIDE_OPERATOR_IMAGE"
		LocalOperandImageVar  = "CLUSTERRESOURCEOVERRIDE_OPERAND_IMAGE"
	)

	operator := os.Getenv(LocalOperatorImageVar)
	if operator == "" {
		err = fmt.Errorf("ENV var %s not defined", LocalOperatorImageVar)
		return
	}

	operand := os.Getenv(LocalOperandImageVar)
	if operator == "" {
		err = fmt.Errorf("ENV var %s not defined", LocalOperandImageVar)
		return
	}

	image = &Image{
		Operator: operator,
		Operand:  operand,
	}

	return
}

func write(image *Image, output string) error {
	bytes, err := json.Marshal(image)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(output, bytes, 0644)
}
