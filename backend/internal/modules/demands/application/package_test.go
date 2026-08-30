package application

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	pdfapi "github.com/pdfcpu/pdfcpu/pkg/api"

	"github.com/yurythx/projeto-nova/internal/modules/demands/domain"
)

// realPDF renderiza um PDF oficial de verdade (fpdf) e devolve seus bytes —
// insumo para os testes do PDF unificado, que precisam de PDFs válidos para
// o pdfcpu concatenar.
func realPDF(t *testing.T) []byte {
	t.Helper()
	d := domain.MonthlyDemand{
		ID: uuid.New(), ContratoNumero: "1/2026", AnoMes: "2026-08",
		Etapa: domain.Etapa1ElaborarOF, Contratado: "ACME LTDA",
	}
	var buf bytes.Buffer
	if err := RenderDemandPDF(PDFOficio, d, &buf); err != nil {
		t.Fatalf("RenderDemandPDF: %v", err)
	}
	return buf.Bytes()
}

// fakeGetter serve bytes canônicos por object path; um path em `missing`
// devolve erro (simula anexo sumido do bucket).
type fakeGetter struct {
	blobs   map[string]string
	missing map[string]bool
}

func (f fakeGetter) Get(_ context.Context, _, obj string) (io.ReadCloser, error) {
	if f.missing[obj] {
		return nil, fmt.Errorf("NoSuchKey")
	}
	b, ok := f.blobs[obj]
	if !ok {
		return nil, fmt.Errorf("NoSuchKey")
	}
	return io.NopCloser(strings.NewReader(b)), nil
}

func readZip(t *testing.T, b []byte) map[string]string {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		t.Fatalf("zip inválido: %v", err)
	}
	out := map[string]string{}
	for _, f := range zr.File {
		rc, _ := f.Open()
		data, _ := io.ReadAll(rc)
		rc.Close()
		out[f.Name] = string(data)
	}
	return out
}

func pkgDoc(dt domain.DocumentType, file, objPath string, uploaded time.Time) domain.DemandDocument {
	return domain.DemandDocument{
		ID: uuid.New(), DemandaID: uuid.New(), DocType: dt,
		FileName: file, FilePath: objPath, UploadedAt: uploaded,
	}
}

func TestWritePackageZip_OrdersByWorkflowAndIncludesIndex(t *testing.T) {
	now := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
	d := domain.MonthlyDemand{
		ID: uuid.New(), ContratoNumero: "012/2026", AnoMes: "2026-08",
		Etapa: domain.Etapa3EmitirOS, Contratado: "ACME LTDA",
		Documents: []domain.DemandDocument{
			// fora de ordem de propósito
			pkgDoc(domain.DocEmpenhoAssinado, "empenho.pdf", "obj/empenho.pdf", now),
			pkgDoc(domain.DocOFPreEmpenho, "of.pdf", "obj/of.pdf", now),
			pkgDoc(domain.DocOficioPlanej, "oficio.pdf", "obj/oficio.pdf", now),
		},
	}
	g := fakeGetter{blobs: map[string]string{
		"obj/empenho.pdf": "EMPENHO-BYTES",
		"obj/of.pdf":      "OF-BYTES",
		"obj/oficio.pdf":  "OFICIO-BYTES",
	}}

	var buf bytes.Buffer
	if err := writePackageZip(context.Background(), g, "bkt", d, &buf, nil, nil); err != nil {
		t.Fatalf("writePackageZip: %v", err)
	}

	files := readZip(t, buf.Bytes())
	var names []string
	for n := range files {
		names = append(names, n)
	}

	// OF antes de Ofício antes de Empenho, com prefixo 01/02/03.
	if _, ok := files["01 - Ordem-de-Fornecimento-Pre-Empenho.pdf"]; !ok {
		t.Errorf("esperava 01 = OF; nomes=%v", names)
	}
	if files["01 - Ordem-de-Fornecimento-Pre-Empenho.pdf"] != "OF-BYTES" {
		t.Errorf("conteúdo do OF errado: %q", files["01 - Ordem-de-Fornecimento-Pre-Empenho.pdf"])
	}
	if _, ok := files["03 - Nota-de-Empenho-Assinada.pdf"]; !ok {
		t.Errorf("esperava 03 = Empenho; nomes=%v", names)
	}
	idx, ok := files["00 - INDICE.txt"]
	if !ok {
		t.Fatalf("faltou 00 - INDICE.txt; nomes=%v", names)
	}
	for _, want := range []string{"Contrato: 012/2026", "Competencia: 2026-08", "ACME LTDA", "ARQUIVOS (3)"} {
		if !strings.Contains(idx, want) {
			t.Errorf("índice sem %q:\n%s", want, idx)
		}
	}
	if _, ok := files["99 - ARQUIVOS-COM-ERRO.txt"]; ok {
		t.Errorf("não deveria haver arquivo de erro quando tudo baixa")
	}
}

func TestWritePackageZip_MissingBlobGoesToErrorFileNotFatal(t *testing.T) {
	now := time.Now()
	d := domain.MonthlyDemand{
		ID: uuid.New(), ContratoNumero: "1/26", AnoMes: "2026-08",
		Documents: []domain.DemandDocument{
			pkgDoc(domain.DocOFPreEmpenho, "of.pdf", "obj/of.pdf", now),
			pkgDoc(domain.DocOficioPlanej, "sumiu.pdf", "obj/sumiu.pdf", now),
		},
	}
	g := fakeGetter{
		blobs:   map[string]string{"obj/of.pdf": "OF"},
		missing: map[string]bool{"obj/sumiu.pdf": true},
	}

	var buf bytes.Buffer
	if err := writePackageZip(context.Background(), g, "bkt", d, &buf, nil, nil); err != nil {
		t.Fatalf("um anexo ausente não pode derrubar o pacote: %v", err)
	}
	files := readZip(t, buf.Bytes())
	if files["01 - Ordem-de-Fornecimento-Pre-Empenho.pdf"] != "OF" {
		t.Errorf("o anexo bom deveria estar presente; nomes=%v", files)
	}
	errfile, ok := files["99 - ARQUIVOS-COM-ERRO.txt"]
	if !ok || !strings.Contains(errfile, "obj/sumiu.pdf") {
		t.Errorf("esperava o anexo ausente listado em 99 - ARQUIVOS-COM-ERRO.txt: %q", errfile)
	}
}

func TestWritePackageZip_IncludesGeneratedDocs(t *testing.T) {
	d := domain.MonthlyDemand{
		ID: uuid.New(), ContratoNumero: "5/26", AnoMes: "2026-08",
		Documents: []domain.DemandDocument{pkgDoc(domain.DocOFPreEmpenho, "of.pdf", "obj/of.pdf", time.Now())},
	}
	g := fakeGetter{blobs: map[string]string{"obj/of.pdf": "OF"}}
	gen := []generatedDoc{
		{name: "GERADO - oficio.pdf", render: func(w io.Writer) error { _, err := io.WriteString(w, "%PDF-FAKE"); return err }},
	}

	var buf bytes.Buffer
	if err := writePackageZip(context.Background(), g, "bkt", d, &buf, gen, nil); err != nil {
		t.Fatalf("writePackageZip: %v", err)
	}
	files := readZip(t, buf.Bytes())
	if files["90 - GERADO - oficio.pdf"] != "%PDF-FAKE" {
		t.Errorf("documento gerado ausente/errado; nomes=%v", files)
	}
	if !strings.Contains(files["00 - INDICE.txt"], "GERADO - oficio.pdf") {
		t.Errorf("índice não menciona o documento gerado")
	}
}

func TestWritePackagePDF_MergesAttachmentsAndGeneratedDocs(t *testing.T) {
	pdfBytes := string(realPDF(t))
	d := domain.MonthlyDemand{
		ID: uuid.New(), ContratoNumero: "012/2026", AnoMes: "2026-08",
		Etapa: domain.Etapa3EmitirOS, Contratado: "ACME LTDA",
		Documents: []domain.DemandDocument{
			pkgDoc(domain.DocEmpenhoAssinado, "empenho.pdf", "obj/empenho.pdf", time.Now()),
			pkgDoc(domain.DocOFPreEmpenho, "of.pdf", "obj/of.pdf", time.Now()),
			// anexo não-PDF/imagem: deve ser pulado sem derrubar o merge
			pkgDoc(domain.DocOficioPlanej, "planilha.xlsx", "obj/planilha.xlsx", time.Now()),
		},
	}
	g := fakeGetter{blobs: map[string]string{
		"obj/empenho.pdf":   pdfBytes,
		"obj/of.pdf":        pdfBytes,
		"obj/planilha.xlsx": "PK\x03\x04not-a-pdf",
	}}
	gen := []generatedDoc{
		{name: "GERADO - oficio.pdf", render: func(w io.Writer) error { return RenderDemandPDF(PDFOficio, d, w) }},
	}

	var buf bytes.Buffer
	if err := writePackagePDF(context.Background(), g, "bkt", d, &buf, gen, nil); err != nil {
		t.Fatalf("writePackagePDF: %v", err)
	}
	if !bytes.HasPrefix(buf.Bytes(), []byte("%PDF")) {
		t.Fatalf("saída não é PDF: começa com %q", buf.Bytes()[:min(8, buf.Len())])
	}
	conf := pdfapi.LoadConfiguration()
	if err := pdfapi.Validate(bytes.NewReader(buf.Bytes()), conf); err != nil {
		t.Fatalf("PDF unificado inválido: %v", err)
	}
	n, err := pdfapi.PageCount(bytes.NewReader(buf.Bytes()), conf)
	if err != nil {
		t.Fatalf("PageCount: %v", err)
	}
	// 2 anexos PDF + 1 doc gerado, cada Ofício com >=1 página → >=3.
	if n < 3 {
		t.Errorf("esperava >=3 páginas (2 anexos + 1 gerado), veio %d", n)
	}
}

func TestWritePackagePDF_NoUnifiablePartsReturnsBadRequest(t *testing.T) {
	d := domain.MonthlyDemand{
		ID: uuid.New(), ContratoNumero: "1/26", AnoMes: "2026-08",
		Etapa: 0, // abaixo da etapa 1 → nenhum documento oficial gerado
		Documents: []domain.DemandDocument{
			pkgDoc(domain.DocOFPreEmpenho, "of.txt", "obj/of.txt", time.Now()),
		},
	}
	g := fakeGetter{blobs: map[string]string{"obj/of.txt": "só texto, não é PDF"}}

	var buf bytes.Buffer
	err := writePackagePDF(context.Background(), g, "bkt", d, &buf, nil, nil)
	if err == nil {
		t.Fatal("esperava erro quando nenhum anexo é PDF/imagem")
	}
	if !strings.Contains(err.Error(), "unificar") {
		t.Errorf("erro inesperado: %v", err)
	}
}

func TestPackagePDFFileName(t *testing.T) {
	d := domain.MonthlyDemand{ContratoNumero: "012/2026", AnoMes: "2026-08"}
	if got := PackagePDFFileName(d); got != "demanda-012-2026-2026-08.pdf" {
		t.Errorf("PackagePDFFileName = %q", got)
	}
}

func TestPackageFileName(t *testing.T) {
	d := domain.MonthlyDemand{ContratoNumero: "012/2026", AnoMes: "2026-08"}
	if got := PackageFileName(d); got != "demanda-012-2026-2026-08.zip" {
		t.Errorf("PackageFileName = %q", got)
	}
	// sem número de contrato cai no ID
	id := uuid.New()
	d2 := domain.MonthlyDemand{ContratoID: id, AnoMes: "2026-08"}
	if got := PackageFileName(d2); !strings.Contains(got, id.String()) {
		t.Errorf("PackageFileName sem contrato deveria usar o ID: %q", got)
	}
}
