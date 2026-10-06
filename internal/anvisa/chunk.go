package anvisa

import (
	"sort"

	"github.com/Ntanzi07/PharmaCode-UPX2026/internal/importer"
)

// DefaultChunkSize is how many drugs go in one transaction. The whole base is
// tens of thousands of rows, and a single transaction over all of them would
// hold locks for minutes and could run the server out of memory.
const DefaultChunkSize = 500

// Chunks splits the parsed rows into files that can be applied one at a time.
//
// Each chunk carries a set of drugs together with their own packages, so a
// chunk never depends on another one having been committed first: that is what
// makes --dry-run honest (every chunk is rolled back on its own) and what lets
// an interrupted import be resumed by simply running it again.
//
// Packages whose drug is not in the drugs file go into the trailing chunks.
// Their drug may already be registered, and when it is not the import service
// reports the row instead of failing.
func Chunks(drugs []importer.DrugRow, packages []importer.PackageRow, size int) []*importer.File {
	if size <= 0 {
		size = DefaultChunkSize
	}

	byDrug := make(map[string][]importer.PackageRow, len(drugs))
	for _, p := range packages {
		byDrug[p.DrugRegistration] = append(byDrug[p.DrugRegistration], p)
	}

	files := make([]*importer.File, 0, len(drugs)/size+1)
	for start := 0; start < len(drugs); start += size {
		end := min(start+size, len(drugs))
		file := &importer.File{Drugs: drugs[start:end]}
		for _, d := range file.Drugs {
			if own, ok := byDrug[d.RegistrationNumber]; ok {
				file.Packages = append(file.Packages, own...)
				delete(byDrug, d.RegistrationNumber)
			}
		}
		files = append(files, file)
	}

	// What is left belongs to drugs the file did not bring. Sorted so two runs
	// over the same files produce the same chunks, which makes a failure
	// halfway through reproducible.
	var orphans []importer.PackageRow
	for _, own := range byDrug {
		orphans = append(orphans, own...)
	}
	sort.Slice(orphans, func(i, j int) bool {
		return orphans[i].PresentationRegistration < orphans[j].PresentationRegistration
	})
	for start := 0; start < len(orphans); start += size {
		end := min(start+size, len(orphans))
		files = append(files, &importer.File{Packages: orphans[start:end]})
	}
	return files
}
