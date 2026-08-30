package application

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	pdfapi "github.com/pdfcpu/pdfcpu/pkg/api"
	pdfmodel "github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"

	apperrors "github.com/yurythx/projeto-nova/internal/domain/errors"
	"github.com/yurythx/projeto-nova/internal/modules/demands/domain"
)

// documentWorkflowOrder fixa a ordem canônica dos documentos dentro do
// pacote (a mesma sequência das 6 etapas): OF/Pré-Empenho → Ofício →
// Empenho → Comprovante → NF → Recepção → Extrato → Relatório → 6 certidões
// → DAM/Medição. Um tipo fora desta lista vai para o fim.
var documentWorkflowOrder = []domain.DocumentType{
	domain.DocOFPreEmpenho, domain.DocOficioPlanej,
	domain.DocEmpenhoAssinado, domain.DocComprovanteEnvio,
	domain.DocNotaFiscal, domain.DocOrdemRecepcao,
	domain.DocExtratoEmpenho, domain.DocRelatorioPgto,
	domain.DocCertidaoSimples, domain.DocCertidaoCNDT, domain.DocCertidaoFGTS,
	domain.DocCertidaoMunicipal, domain.DocCertidaoEstadual, domain.DocCertidaoFederal,
	domain.DocGuiaDAMISSQN, domain.DocPlanilhaMedicao,
}

func docSortIndex(d domain.DocumentType) int {
	for i, t := range documentWorkflowOrder {
		if t == d {
			return i
		}
	}
	return len(documentWorkflowOrder) + 1
}

var (
	slugAccents = strings.NewReplacer(
		"á", "a", "à", "a", "â", "a", "ã", "a", "ä", "a",
		"é", "e", "ê", "e", "è", "e", "ë", "e",
		"í", "i", "ì", "i", "î", "i", "ï", "i",
		"ó", "o", "ò", "o", "ô", "o", "õ", "o", "ö", "o",
		"ú", "u", "ù", "u", "û", "u", "ü", "u",
		"ç", "c", "ñ", "n",
		"Á", "A", "À", "A", "Â", "A", "Ã", "A",
		"É", "E", "Ê", "E", "Í", "I",
		"Ó", "O", "Ô", "O", "Õ", "O", "Ú", "U", "Ç", "C",
	)
	slugNonWord   = regexp.MustCompile(`[^0-9A-Za-z]+`)
	slugMultiDash = regexp.MustCompile(`-{2,}`)
)

// slugify normaliza um texto livre para nome de arquivo seguro: remove
// acentos, troca qualquer run de não-alfanumérico por "-" e enxuga hifens.
func slugify(s string) string {
	s = slugAccents.Replace(strings.TrimSpace(s))
	s = slugNonWord.ReplaceAllString(s, "-")
	s = slugMultiDash.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		return "sem-nome"
	}
	return s
}

// packageBaseName é "demanda-<contrato>-<ano_mes>" — o tronco comum dos
// nomes de arquivo do .zip e do .pdf unificado.
func packageBaseName(d domain.MonthlyDemand) string {
	num := d.ContratoNumero
	if num == "" {
		num = d.ContratoID.String()
	}
	return fmt.Sprintf("demanda-%s-%s", slugify(num), slugify(d.AnoMes))
}

// PackageFileName devolve o nome sugerido do .zip de uma demanda
// (demanda-<contrato>-<ano_mes>.zip), sem tocar no storage.
func PackageFileName(d domain.MonthlyDemand) string {
	return packageBaseName(d) + ".zip"
}

// PackagePDFFileName devolve o nome sugerido do PDF unificado de uma demanda
// (demanda-<contrato>-<ano_mes>.pdf).
func PackagePDFFileName(d domain.MonthlyDemand) string {
	return packageBaseName(d) + ".pdf"
}

// blobGetter é o subconjunto de storage.Provider que o compilador de pacote
// usa — deixa o teste injetar um fake sem MinIO.
type blobGetter interface {
	Get(ctx context.Context, bucketName, objectName string) (io.ReadCloser, error)
}

// readObject lê um objeto inteiro para memória, fechando o reader. Retorna
// erro tanto na abertura quanto na leitura (o cliente MinIO só falha no Read).
func readObject(ctx context.Context, getter blobGetter, bucket, key string) ([]byte, error) {
	rc, err := getter.Get(ctx, bucket, key)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

// WriteDemandPackage baixa do storage todos os anexos da demanda `d` e os
// escreve num único .zip em `out`, nomeados com prefixo numérico na ordem
// das etapas (ex.: "01 - Ordem-de-Fornecimento...pdf") e precedidos de um
// "00 - INDICE.txt". Um anexo que não abre no storage não derruba o pacote:
// entra em "99 - ARQUIVOS-COM-ERRO.txt".
//
// A demanda precisa vir com Documents carregados; use LoadDemandForPackage
// antes, na camada HTTP, para responder 400 quando não há nada a compactar.
func (s *Service) WriteDemandPackage(ctx context.Context, d domain.MonthlyDemand, out io.Writer) error {
	return writePackageZip(ctx, s.storage, s.bucket, d, out, generatedDemandDocs(d), s.logger)
}

// generatedDoc é um documento oficial renderizado em tempo real (não vem do
// storage) para entrar no pacote (.zip ou PDF unificado).
type generatedDoc struct {
	name   string
	render func(io.Writer) error
}

// generatedDemandDocs lista os documentos oficiais gerados na hora conforme a
// etapa alcançada: o Ofício sempre; a OS a partir da etapa 3; o Anexo I a
// partir da 5. Mesma sequência para o .zip e para o PDF unificado.
func generatedDemandDocs(d domain.MonthlyDemand) []generatedDoc {
	var generated []generatedDoc
	for _, k := range []PDFKind{PDFOficio, PDFOrdemServico, PDFRelatorio} {
		if d.Etapa < pdfKindMinEtapa[k] {
			continue
		}
		kind := k
		generated = append(generated, generatedDoc{
			name:   "GERADO - " + string(kind) + ".pdf",
			render: func(w io.Writer) error { return RenderDemandPDF(kind, d, w) },
		})
	}
	return generated
}

// sortedDemandDocs devolve os anexos de `d` na ordem canônica das etapas
// (documentWorkflowOrder), desempatando por data de upload.
func sortedDemandDocs(d domain.MonthlyDemand) []domain.DemandDocument {
	docs := append([]domain.DemandDocument(nil), d.Documents...)
	sort.SliceStable(docs, func(i, j int) bool {
		si, sj := docSortIndex(docs[i].DocType), docSortIndex(docs[j].DocType)
		if si != sj {
			return si < sj
		}
		return docs[i].UploadedAt.Before(docs[j].UploadedAt)
	})
	return docs
}

func writePackageZip(ctx context.Context, getter blobGetter, bucket string, d domain.MonthlyDemand, out io.Writer, generated []generatedDoc, logger *slog.Logger) error {
	docs := sortedDemandDocs(d)

	zw := zip.NewWriter(out)

	var index strings.Builder
	fmt.Fprintf(&index, "PACOTE DA DEMANDA MENSAL\r\n")
	fmt.Fprintf(&index, "Contrato: %s\r\n", d.ContratoNumero)
	if d.Contratado != "" {
		fmt.Fprintf(&index, "Contratado: %s\r\n", d.Contratado)
	}
	fmt.Fprintf(&index, "Competencia: %s\r\n", d.AnoMes)
	fmt.Fprintf(&index, "Etapa atual: %d\r\n", d.Etapa)
	fmt.Fprintf(&index, "Gerado em: %s\r\n\r\n", time.Now().Format("02/01/2006 15:04"))
	fmt.Fprintf(&index, "ARQUIVOS (%d):\r\n", len(docs))

	var failed []string
	seen := map[string]int{}

	for i, doc := range docs {
		ext := path.Ext(doc.FileName)
		base := fmt.Sprintf("%02d - %s", i+1, slugify(doc.DocType.Label()))
		name := base + ext
		if n := seen[base+ext]; n > 0 {
			name = fmt.Sprintf("%s (%d)%s", base, n+1, ext)
		}
		seen[base+ext]++

		// O cliente MinIO faz GetObject preguiçoso: o erro (objeto ausente
		// no bucket, ex. um upload presigned que falhou no navegador) só
		// aparece no primeiro Read. Por isso lemos TUDO para um buffer
		// antes de criar a entrada no zip — assim um anexo ilegível vira
		// uma linha em 99 - ARQUIVOS-COM-ERRO.txt em vez de abortar (e
		// corromper) o pacote inteiro. Anexos são PDFs/imagens pequenos.
		content, err := readObject(ctx, getter, bucket, doc.FilePath)
		if err != nil {
			failed = append(failed, fmt.Sprintf("%s — %s (%s): %v", name, doc.DocType.Label(), doc.FilePath, err))
			fmt.Fprintf(&index, "  [FALHOU] %s\r\n", name)
			continue
		}

		validade := ""
		if doc.Validade != nil {
			validade = " | validade " + doc.Validade.Format("02/01/2006")
		}
		fmt.Fprintf(&index, "  %s  (enviado %s%s)\r\n", name, doc.UploadedAt.Format("02/01/2006"), validade)

		w, err := zw.Create(name)
		if err != nil {
			_ = zw.Close()
			return fmt.Errorf("demands package: criar entrada zip %q: %w", name, err)
		}
		if _, err := w.Write(content); err != nil {
			_ = zw.Close()
			return fmt.Errorf("demands package: escrever %q: %w", name, err)
		}
	}

	// Documentos oficiais gerados (Ofício / OS / Anexo I) — prefixo alto
	// para ordenarem depois dos anexos.
	for i, g := range generated {
		name := fmt.Sprintf("%02d - %s", 90+i, g.name)
		w, err := zw.Create(name)
		if err != nil {
			_ = zw.Close()
			return fmt.Errorf("demands package: criar entrada zip %q: %w", name, err)
		}
		if err := g.render(w); err != nil {
			failed = append(failed, fmt.Sprintf("%s: %v", name, err))
			fmt.Fprintf(&index, "  [FALHOU] %s\r\n", name)
			continue
		}
		fmt.Fprintf(&index, "  %s  (gerado automaticamente)\r\n", name)
	}

	if hw, err := zw.Create("00 - INDICE.txt"); err == nil {
		_, _ = io.WriteString(hw, index.String())
	}
	if len(failed) > 0 {
		if ew, err := zw.Create("99 - ARQUIVOS-COM-ERRO.txt"); err == nil {
			_, _ = io.WriteString(ew, strings.Join(failed, "\r\n")+"\r\n")
		}
		if logger != nil {
			logger.Warn("demands package: anexos ausentes no storage",
				"demanda_id", d.ID.String(), "faltando", len(failed))
		}
	}

	if err := zw.Close(); err != nil {
		return fmt.Errorf("demands package: fechar zip: %w", err)
	}
	return nil
}

// WriteDemandPackagePDF baixa os anexos da demanda `d`, gera os documentos
// oficiais devidos à etapa (Ofício / OS / Anexo I) e concatena TUDO num
// único PDF em `out`, na ordem canônica das etapas — o maço pronto para
// despacho ao fornecedor. Diferente do .zip, não há índice nem errata
// embutidos: é só o PDF unificado.
//
// Cada anexo é lido inteiro para memória (mesmo motivo do .zip: o cliente
// MinIO só falha no primeiro Read). Anexos que já são PDF entram como estão;
// imagens (JPEG/PNG) viram uma página cada via pdfcpu; qualquer outro tipo
// (ou um anexo ilegível no storage) é pulado e registrado no log — nunca
// derruba o pacote. Erra com BadRequest se, no fim, não sobrar nada
// unificável.
func (s *Service) WriteDemandPackagePDF(ctx context.Context, d domain.MonthlyDemand, out io.Writer) error {
	return writePackagePDF(ctx, s.storage, s.bucket, d, out, generatedDemandDocs(d), s.logger)
}

func writePackagePDF(ctx context.Context, getter blobGetter, bucket string, d domain.MonthlyDemand, out io.Writer, generated []generatedDoc, logger *slog.Logger) error {
	parts := make([]io.ReadSeeker, 0, len(d.Documents)+3)
	var skipped []string

	for i, doc := range sortedDemandDocs(d) {
		label := fmt.Sprintf("%02d - %s", i+1, doc.DocType.Label())
		content, err := readObject(ctx, getter, bucket, doc.FilePath)
		if err != nil {
			skipped = append(skipped, fmt.Sprintf("%s (%s): %v", label, doc.FilePath, err))
			continue
		}
		switch {
		case bytes.HasPrefix(content, []byte("%PDF")):
			parts = append(parts, bytes.NewReader(content))
		case isSupportedImage(content):
			var page bytes.Buffer
			if err := pdfapi.ImportImages(nil, &page, []io.Reader{bytes.NewReader(content)}, nil, nil); err != nil {
				skipped = append(skipped, fmt.Sprintf("%s: imagem não convertida: %v", label, err))
				continue
			}
			parts = append(parts, bytes.NewReader(page.Bytes()))
		default:
			skipped = append(skipped, fmt.Sprintf("%s: tipo não suportado no PDF unificado", label))
		}
	}

	for _, g := range generated {
		var buf bytes.Buffer
		if err := g.render(&buf); err != nil {
			skipped = append(skipped, fmt.Sprintf("%s: %v", g.name, err))
			continue
		}
		parts = append(parts, bytes.NewReader(buf.Bytes()))
	}

	if logger != nil && len(skipped) > 0 {
		logger.Warn("demands package pdf: itens fora do PDF unificado",
			"demanda_id", d.ID.String(), "pulados", len(skipped))
	}
	if len(parts) == 0 {
		return apperrors.BadRequest("nenhum anexo em PDF ou imagem para unificar num só PDF")
	}

	conf := pdfmodel.NewDefaultConfiguration()
	conf.ValidationMode = pdfmodel.ValidationRelaxed
	if err := pdfapi.MergeRaw(parts, out, false, conf); err != nil {
		return fmt.Errorf("demands package pdf: merge: %w", err)
	}
	return nil
}

// isSupportedImage sniffa a assinatura de bytes de JPEG e PNG — os formatos
// que pdfcpu.ImportImages aceita e que aparecem como anexo (certidões
// escaneadas, prints).
func isSupportedImage(b []byte) bool {
	return bytes.HasPrefix(b, []byte{0xFF, 0xD8, 0xFF}) || // JPEG
		bytes.HasPrefix(b, []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}) // PNG
}

// LoadDemandForPDF busca a demanda, valida que ela alcançou a etapa mínima
// do documento `kind` e devolve o nome de arquivo sugerido — tudo ANTES de
// a camada HTTP escrever qualquer cabeçalho, para poder responder 400/404
// em JSON em vez de um PDF quebrado. Depois disso a camada HTTP chama
// RenderDemandPDF(kind, demand, w).
func (s *Service) LoadDemandForPDF(ctx context.Context, demandID uuid.UUID, kind PDFKind) (domain.MonthlyDemand, string, error) {
	d, err := s.repo.GetByID(ctx, demandID)
	if err != nil {
		return domain.MonthlyDemand{}, "", fmt.Errorf("demands pdf: get demand: %w", err)
	}
	minEtapa, ok := pdfKindMinEtapa[kind]
	if !ok {
		return domain.MonthlyDemand{}, "", apperrors.BadRequest("tipo de documento desconhecido: " + string(kind))
	}
	if d.Etapa < minEtapa {
		return domain.MonthlyDemand{}, "", apperrors.BadRequest(fmt.Sprintf(
			"o documento só é emitido a partir da etapa %d (a demanda está na %d)", minEtapa, d.Etapa))
	}
	return d, PDFFileName(kind, d), nil
}

// LoadDemandForPackage busca a demanda e valida que há ao menos um anexo
// para compactar — a camada HTTP chama isto ANTES de escrever cabeçalhos,
// para poder responder 404/400 em vez de um .zip quebrado.
func (s *Service) LoadDemandForPackage(ctx context.Context, demandID uuid.UUID) (domain.MonthlyDemand, error) {
	d, err := s.repo.GetByID(ctx, demandID)
	if err != nil {
		return domain.MonthlyDemand{}, fmt.Errorf("demands package: get demand: %w", err)
	}
	if len(d.Documents) == 0 {
		return domain.MonthlyDemand{}, apperrors.BadRequest("a demanda não tem nenhum documento anexado para compactar")
	}
	return d, nil
}
