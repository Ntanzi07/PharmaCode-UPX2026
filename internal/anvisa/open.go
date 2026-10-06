package anvisa

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

// Open reads a source given as a local path or as an http(s) URL, and always
// returns semicolon separated text, converting a spreadsheet on the way: the
// CMED list is published both as .csv (open data) and as .xlsx (gov.br), and
// the parsers only have to know about one of them.
//
// The caller closes the returned reader.
func Open(source string) (io.ReadCloser, error) {
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		return download(source)
	}
	f, err := os.Open(source)
	if err != nil {
		return nil, err
	}
	return tabular(f)
}

// download fetches the file. Anvisa answers 403 to a client without a browser
// User-Agent, so one is sent.
func download(url string) (io.ReadCloser, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; PharmaCode/1.0; +https://github.com/Ntanzi07/PharmaCode-UPX2026)")
	req.Header.Set("Accept", "*/*")

	client := &http.Client{Timeout: 30 * time.Minute}
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not download %s: %w", url, err)
	}
	if res.StatusCode != http.StatusOK {
		res.Body.Close()
		return nil, fmt.Errorf("%s answered %s", url, res.Status)
	}
	return tabular(res.Body)
}

// zipMagic are the first bytes of every .xlsx (which is a zip).
var zipMagic = []byte("PK\x03\x04")

// tabular passes a CSV straight through and converts a spreadsheet, detecting
// which one it is by the first bytes instead of by the file extension.
func tabular(r io.ReadCloser) (io.ReadCloser, error) {
	br := bufio.NewReaderSize(r, 32*1024)
	head, _ := br.Peek(len(zipMagic))
	if !bytes.Equal(head, zipMagic) {
		return readCloser{Reader: br, Closer: r}, nil
	}

	// excelize needs to seek, so the spreadsheet is read into memory. The CMED
	// list is around 10 MB, which is fine; a CSV is never loaded this way.
	data, err := io.ReadAll(br)
	r.Close()
	if err != nil {
		return nil, err
	}
	return xlsxToCSV(data)
}

// xlsxToCSV turns the first sheet into semicolon separated text, written in the
// background so a big spreadsheet is not held twice in memory.
func xlsxToCSV(data []byte) (io.ReadCloser, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("could not read the spreadsheet: %w", err)
	}
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		f.Close()
		return nil, fmt.Errorf("the spreadsheet has no sheets")
	}
	rows, err := f.Rows(sheets[0])
	if err != nil {
		f.Close()
		return nil, err
	}

	pr, pw := io.Pipe()
	go func() {
		defer f.Close()
		defer rows.Close()

		w := csv.NewWriter(pw)
		w.Comma = ';'
		for rows.Next() {
			cells, err := rows.Columns()
			if err != nil {
				pw.CloseWithError(err)
				return
			}
			if err := w.Write(cells); err != nil {
				pw.CloseWithError(err)
				return
			}
		}
		w.Flush()
		pw.CloseWithError(w.Error())
	}()
	return pr, nil
}

// readCloser keeps the buffered reader while closing the original source.
type readCloser struct {
	io.Reader
	io.Closer
}
