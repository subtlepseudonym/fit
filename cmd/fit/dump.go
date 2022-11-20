package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	fit "github.com/subtlepseudonym/fit-go"

	"github.com/spf13/cobra"
)

func NewDumpCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "dump",
		Short: "Dump file header and ID",
		RunE:  dump,
	}
}

func dump(cmd *cobra.Command, args []string) error {
	for _, arg := range args {
		f, err := os.Open(arg)
		if err != nil {
			return fmt.Errorf("open: %w", err)
		}
		defer f.Close()

		header, fileID, err := fit.DecodeHeaderAndFileID(f)
		if err != nil {
			return fmt.Errorf("decode header and file ID: %w", err)
		}

		b, err := json.Marshal(header)
		if err != nil {
			return fmt.Errorf("marshal header: %w", err)
		}
		fmt.Println(string(b))

		fid := struct {
			Type         string
			Manufacturer string
			Product      interface{}
			SerialNumber uint32
			TimeCreated  time.Time
			Number       uint16
			ProductName  string
		}{
			Type:         fileID.Type.String(),
			Manufacturer: fileID.Manufacturer.String(),
			SerialNumber: fileID.SerialNumber,
			TimeCreated:  fileID.TimeCreated,
			Number:       fileID.Number,
			ProductName:  fileID.ProductName,
		}

		product := fileID.GetProduct()
		if p, ok := product.(fit.GarminProduct); ok {
			fid.Product = p.String()
		} else {
			fid.Product = product
		}

		b, err = json.Marshal(fid)
		if err != nil {
			return fmt.Errorf("marshal file ID message: %w", err)
		}
		fmt.Println(string(b))
	}

	return nil
}
