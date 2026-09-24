package handler

import (
	"errors"
	"log"
	"net/http"

	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/importer"
	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/service"
)

// maxUploadBytes is the biggest spreadsheet the API accepts (8 MB).
const maxUploadBytes = 8 << 20

type ImportHandler struct {
	service *service.ImportService
}

func NewImportHandler(s *service.ImportService) *ImportHandler {
	return &ImportHandler{service: s}
}

// DownloadTemplate godoc
// @Summary      Download the spreadsheet template
// @Description  .xlsx with the "remedios" and "embalagens" sheets, an example row and a sheet explaining every column.
// @Tags         imports
// @Produce      application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Success      200  {file}    binary
// @Failure      401  {string}  string  "authentication required"
// @Router       /imports/template [get]
func (h *ImportHandler) DownloadTemplate(w http.ResponseWriter, r *http.Request) {
	file, err := importer.BuildTemplate()
	if err != nil {
		log.Printf("failed to build the import template: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="`+importer.TemplateFileName+`"`)
	if _, err := w.Write(file); err != nil {
		log.Printf("failed to send the import template: %v", err)
	}
}

// PreviewImport godoc
// @Summary      Check a spreadsheet without saving anything
// @Description  Reads the file, validates every row and runs the import inside a transaction that is rolled back. The counts are real; the database is untouched.
// @Tags         imports
// @Accept       multipart/form-data
// @Produce      json
// @Param        file  formData  file  true  "Spreadsheet filled in from the template (.xlsx)"
// @Success      200   {object}  service.ImportResult
// @Failure      400   {string}  string  "file missing, too big or not a valid spreadsheet"
// @Failure      401   {string}  string  "authentication required"
// @Router       /imports/preview [post]
func (h *ImportHandler) PreviewImport(w http.ResponseWriter, r *http.Request) {
	h.run(w, r, true)
}

// ApplyImport godoc
// @Summary      Import a spreadsheet
// @Description  Same checks as the preview, and then saves everything in one transaction. Rows are matched by registration number, presentation registration and EAN, so importing the same file again updates instead of duplicating. Imported leaflets come in as not reviewed.
// @Tags         imports
// @Accept       multipart/form-data
// @Produce      json
// @Param        file  formData  file  true  "Spreadsheet filled in from the template (.xlsx)"
// @Success      200   {object}  service.ImportResult
// @Failure      400   {string}  string  "file missing, too big or not a valid spreadsheet"
// @Failure      401   {string}  string  "authentication required"
// @Failure      422   {object}  service.ImportResult  "the spreadsheet has errors; nothing was saved"
// @Router       /imports/apply [post]
func (h *ImportHandler) ApplyImport(w http.ResponseWriter, r *http.Request) {
	h.run(w, r, false)
}

// run is the shared part of preview and apply.
func (h *ImportHandler) run(w http.ResponseWriter, r *http.Request, dryRun bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)

	file, _, err := r.FormFile("file")
	if err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			http.Error(w, "the spreadsheet is bigger than 8 MB", http.StatusBadRequest)
			return
		}
		http.Error(w, `send the spreadsheet in the "file" field`, http.StatusBadRequest)
		return
	}
	defer file.Close()

	parsed, rowErrors, err := importer.Parse(file)
	if err != nil {
		// The file itself is unusable (not an .xlsx, missing sheet, no rows).
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if len(rowErrors) > 0 {
		result := service.ImportResult{Errors: rowErrors}
		status := http.StatusOK // a preview with errors is a normal answer
		if !dryRun {
			status = http.StatusUnprocessableEntity
		}
		writeJSON(w, status, result)
		return
	}

	result, err := h.service.Run(r.Context(), parsed, dryRun)
	if err != nil {
		log.Printf("failed to import spreadsheet: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if !dryRun && !result.Applied {
		writeJSON(w, http.StatusUnprocessableEntity, result)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
