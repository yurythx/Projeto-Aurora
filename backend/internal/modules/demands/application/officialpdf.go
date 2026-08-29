package application

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"

	apperrors "github.com/yurythx/projeto-nova/internal/domain/errors"
	"github.com/yurythx/projeto-nova/internal/modules/demands/domain"
)

// officialPDF é a casca comum dos documentos oficiais imprimíveis das
// demandas (mesma identidade visual das páginas /contratos/demandas/[id]/*):
// cabeçalho da PMR, caixa de dados do contrato, seções, tabelas simples,
// bloco de assinaturas e rodapé com código de autenticidade.
type officialPDF struct {
	pdf *fpdf.Fpdf
	tr  func(string) string
}

const pdfContentWidth = 170.0 // A4 210mm - margens 20+20

func newOfficialPDF(docTitle, docTag string, d domain.MonthlyDemand) *officialPDF {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(20, 18, 20)
	pdf.SetAutoPageBreak(true, 20)
	p := &officialPDF{pdf: pdf, tr: pdf.UnicodeTranslatorFromDescriptor("")}
	pdf.AddPage()

	// Cabeçalho
	pdf.SetFont("Helvetica", "B", 11)
	pdf.SetTextColor(20, 20, 20)
	pdf.CellFormat(0, 6, p.tr("PREFEITURA MUNICIPAL DE RONDONÓPOLIS"), "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 8.5)
	pdf.SetTextColor(90, 90, 90)
	pdf.CellFormat(0, 4.5, p.tr("Secretaria Municipal de Administração e Gestão de Contratos"), "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 4.5, p.tr("Instrução Normativa SCL nº 01/2019 • Sistema de Controle Interno"), "", 1, "L", false, 0, "")

	y := pdf.GetY() + 1
	pdf.SetDrawColor(20, 20, 20)
	pdf.SetLineWidth(0.4)
	pdf.Line(20, y, 190, y)
	pdf.Ln(4)

	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(90, 90, 90)
	ref := fmt.Sprintf("%s Nº %s/%s", docTag, strings.ToUpper(d.ID.String()[:8]), safeYear(d.AnoMes))
	pdf.CellFormat(0, 4, p.tr(fmt.Sprintf("%s   |   Competência: %s   |   Emitido em: %s",
		ref, d.AnoMes, time.Now().Format("02/01/2006 15:04"))), "", 1, "R", false, 0, "")
	pdf.Ln(3)

	// Título
	pdf.SetFont("Helvetica", "B", 12)
	pdf.SetTextColor(20, 20, 20)
	pdf.MultiCell(0, 6, p.tr(strings.ToUpper(docTitle)), "", "C", false)
	pdf.Ln(3)

	return p
}

func safeYear(anoMes string) string {
	if len(anoMes) >= 4 {
		return anoMes[:4]
	}
	return time.Now().Format("2006")
}

func (p *officialPDF) contractBox(d domain.MonthlyDemand) {
	pdf := p.pdf
	pdf.SetFillColor(246, 246, 246)
	pdf.SetDrawColor(200, 200, 200)
	pdf.SetLineWidth(0.2)
	// Se a caixa não couber inteira na página atual, quebra ANTES de
	// começar — senão o MultiCell interno dispara auto page-break no meio e
	// endY-startY vira negativo (Rect malformado).
	if _, pageH := pdf.GetPageSize(); pdf.GetY()+30 > pageH-20 {
		pdf.AddPage()
	}
	startY := pdf.GetY()
	pdf.SetFont("Helvetica", "", 8.5)
	pdf.SetTextColor(40, 40, 40)

	line := func(label, value string) {
		pdf.SetX(22)
		pdf.SetFont("Helvetica", "B", 8)
		pdf.CellFormat(38, 5, p.tr(label), "", 0, "L", false, 0, "")
		pdf.SetFont("Helvetica", "", 8.5)
		pdf.MultiCell(120, 5, p.tr(value), "", "L", false)
	}
	pdf.Ln(1)
	line("Nº do Contrato:", orDash(d.ContratoNumero))
	line("Contratada:", orDash(d.Contratado))
	line("Objeto:", orDash(d.ContratoObjeto))
	if d.ContratoValor != nil {
		line("Valor / Teto Mensal:", fmt.Sprintf("R$ %s", brl(*d.ContratoValor)))
	}
	pdf.Ln(1)
	if endY := pdf.GetY(); endY > startY {
		pdf.Rect(20, startY, pdfContentWidth, endY-startY, "D")
	}
	pdf.Ln(3)
}

func (p *officialPDF) section(title string) {
	pdf := p.pdf
	pdf.Ln(1)
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetTextColor(20, 20, 20)
	pdf.CellFormat(0, 5, p.tr(strings.ToUpper(title)), "B", 1, "L", false, 0, "")
	pdf.Ln(1.5)
}

func (p *officialPDF) paragraph(text string) {
	pdf := p.pdf
	pdf.SetFont("Helvetica", "", 8.5)
	pdf.SetTextColor(40, 40, 40)
	pdf.MultiCell(0, 4.6, p.tr(text), "", "J", false)
	pdf.Ln(2)
}

func (p *officialPDF) table(headers []string, widths []float64, rows [][]string) {
	pdf := p.pdf
	pdf.SetFont("Helvetica", "B", 7.5)
	pdf.SetFillColor(235, 235, 235)
	pdf.SetTextColor(30, 30, 30)
	for i, h := range headers {
		pdf.CellFormat(widths[i], 6, p.tr(h), "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)
	pdf.SetFont("Helvetica", "", 7.5)
	_, pageH := pdf.GetPageSize()
	for _, row := range rows {
		// altura da linha = maior célula multilinha
		h := 5.0
		for i, cell := range row {
			lines := pdf.SplitLines([]byte(p.tr(cell)), widths[i]-2)
			if lh := float64(len(lines)) * 4.2; lh > h {
				h = lh
			}
		}
		// As células da linha são desenhadas com Rect() manual, que não
		// dispara o auto page-break do fpdf — força a quebra aqui se a
		// linha não couber na página atual (margem inferior de 20mm).
		if pdf.GetY()+h > pageH-20 {
			pdf.AddPage()
			pdf.SetFont("Helvetica", "B", 7.5)
			pdf.SetFillColor(235, 235, 235)
			for i, hd := range headers {
				pdf.CellFormat(widths[i], 6, p.tr(hd), "1", 0, "C", true, 0, "")
			}
			pdf.Ln(-1)
			pdf.SetFont("Helvetica", "", 7.5)
		}
		x, y := pdf.GetX(), pdf.GetY()
		for i, cell := range row {
			pdf.Rect(x, y, widths[i], h, "D")
			pdf.MultiCell(widths[i], 4.2, p.tr(cell), "", "L", false)
			x += widths[i]
			pdf.SetXY(x, y)
		}
		pdf.SetXY(pdf.GetX(), y+h)
		pdf.SetX(20)
	}
	pdf.Ln(3)
}

type sigSlot struct{ role, line1, line2 string }

func (p *officialPDF) signatures(slots []sigSlot) {
	pdf := p.pdf
	if _, pageH := pdf.GetPageSize(); pdf.GetY() > pageH-48 {
		pdf.AddPage()
	}
	pdf.Ln(14)
	colW := pdfContentWidth / float64(len(slots))
	yLine := pdf.GetY()
	for i := range slots {
		x := 20 + float64(i)*colW
		pdf.Line(x+8, yLine, x+colW-8, yLine)
	}
	pdf.Ln(1.5)
	for _, s := range slots {
		pdf.SetFont("Helvetica", "B", 8)
		pdf.SetTextColor(20, 20, 20)
		pdf.CellFormat(colW, 4, p.tr(strings.ToUpper(s.role)), "", 0, "C", false, 0, "")
	}
	pdf.Ln(4)
	for _, s := range slots {
		pdf.SetFont("Helvetica", "", 7)
		pdf.SetTextColor(90, 90, 90)
		pdf.CellFormat(colW, 3.5, p.tr(s.line1), "", 0, "C", false, 0, "")
	}
	pdf.Ln(3.5)
	for _, s := range slots {
		pdf.SetFont("Helvetica", "", 6.5)
		pdf.SetTextColor(140, 140, 140)
		pdf.CellFormat(colW, 3, p.tr(s.line2), "", 0, "C", false, 0, "")
	}
	pdf.Ln(6)
}

func (p *officialPDF) finish(d domain.MonthlyDemand, w io.Writer) error {
	pdf := p.pdf
	pdf.SetY(-18)
	pdf.SetFont("Helvetica", "", 6.5)
	pdf.SetTextColor(150, 150, 150)
	pdf.CellFormat(0, 4, p.tr(fmt.Sprintf(
		"Documento gerado automaticamente pelo Projeto Nova • Código de Autenticidade: %s", d.ID.String())),
		"", 1, "C", false, 0, "")
	return pdf.Output(w)
}

func orDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "—"
	}
	return s
}

func brl(v float64) string {
	s := fmt.Sprintf("%.2f", v)
	intPart, dec := s, "00"
	if i := strings.IndexByte(s, '.'); i >= 0 {
		intPart, dec = s[:i], s[i+1:]
	}
	neg := strings.HasPrefix(intPart, "-")
	intPart = strings.TrimPrefix(intPart, "-")
	var out []byte
	for i, c := range []byte(intPart) {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			out = append(out, '.')
		}
		out = append(out, c)
	}
	res := string(out) + "," + dec
	if neg {
		res = "-" + res
	}
	return res
}

// --- Documentos ---

// PDFKind identifica qual documento oficial gerar.
type PDFKind string

const (
	PDFOficio       PDFKind = "oficio"
	PDFOrdemServico PDFKind = "ordem-servico"
	PDFRelatorio    PDFKind = "relatorio"
)

var pdfKindMinEtapa = map[PDFKind]domain.EtapaKanban{
	PDFOficio:       domain.Etapa1ElaborarOF,
	PDFOrdemServico: domain.Etapa3EmitirOS,
	PDFRelatorio:    domain.Etapa5RelatorioPgto,
}

// PDFFileName é o nome sugerido do arquivo para download / entrada no zip.
func PDFFileName(kind PDFKind, d domain.MonthlyDemand) string {
	base := map[PDFKind]string{
		PDFOficio:       "oficio-planejamento",
		PDFOrdemServico: "ordem-servico",
		PDFRelatorio:    "relatorio-fiscalizacao",
	}[kind]
	return fmt.Sprintf("%s-%s-%s.pdf", base, slugify(d.ContratoNumero), slugify(d.AnoMes))
}

// RenderDemandPDF escreve o PDF do documento `kind` da demanda em w.
// apperrors.BadRequest quando a demanda ainda não alcançou a etapa mínima
// em que o documento faz sentido.
func RenderDemandPDF(kind PDFKind, d domain.MonthlyDemand, w io.Writer) error {
	minEtapa, ok := pdfKindMinEtapa[kind]
	if !ok {
		return apperrors.BadRequest("tipo de documento desconhecido: " + string(kind))
	}
	if d.Etapa < minEtapa {
		return apperrors.BadRequest(fmt.Sprintf(
			"o documento só é emitido a partir da etapa %d (demanda está na %d)", minEtapa, d.Etapa))
	}
	switch kind {
	case PDFOficio:
		return renderOficio(d, w)
	case PDFOrdemServico:
		return renderOrdemServico(d, w)
	case PDFRelatorio:
		return renderRelatorio(d, w)
	default:
		return apperrors.BadRequest("tipo de documento desconhecido")
	}
}

func renderOficio(d domain.MonthlyDemand, w io.Writer) error {
	p := newOfficialPDF("Ordem de Fornecimento & Autorização de Pré-Empenho", "OF", d)
	p.contractBox(d)
	p.paragraph(fmt.Sprintf(
		"Pelo presente instrumento, a Secretaria Municipal autoriza a emissão da Nota de Empenho "+
			"referente à demanda mensal do mês de %s, em conformidade com as cláusulas do contrato "+
			"administrativo nº %s.", d.AnoMes, orDash(d.ContratoNumero)))
	p.paragraph("O fornecedor contratado fica notificado a manter a regularidade de todas as certidões " +
		"fiscais e trabalhistas exigidas para a efetivação da liquidação financeira e pagamento.")
	if strings.TrimSpace(d.Observacoes) != "" {
		p.section("Observações da Demanda")
		p.paragraph(d.Observacoes)
	}
	p.section("Documentação anexada")
	p.docList(d)
	p.signatures([]sigSlot{
		{"Fiscal do Contrato", "Portaria de Designação", "Assinatura / Matrícula"},
		{"Ordenador de Despesa / Secretário", "Prefeitura Municipal de Rondonópolis", "Assinatura / Carimbo"},
	})
	return p.finish(d, w)
}

func renderOrdemServico(d domain.MonthlyDemand, w io.Writer) error {
	p := newOfficialPDF("Ordem de Serviço / Autorização de Execução", "OS", d)
	p.contractBox(d)
	p.paragraph(fmt.Sprintf(
		"A Secretaria Municipal, por meio do fiscal do contrato nº %s, AUTORIZA a empresa %s a executar "+
			"os serviços / fornecer os bens referentes à competência %s, nos termos e quantitativos "+
			"pactuados no instrumento contratual e na respectiva Nota de Empenho.",
		orDash(d.ContratoNumero), orDash(d.Contratado), d.AnoMes))

	valor := "—"
	if d.ContratoValor != nil {
		valor = brl(*d.ContratoValor)
	}
	p.table(
		[]string{"Item", "Descrição / Objeto", "Marca", "Qtd.", "Valor Unit.", "Valor Total"},
		[]float64{12, 78, 20, 14, 23, 23},
		[][]string{{"01", orDash(d.ContratoObjeto), "—", "1", valor, valor}},
	)
	p.paragraph("A empresa deverá observar rigorosamente os prazos de entrega/execução e emitir a Nota " +
		"Fiscal correspondente somente após a conclusão, acompanhada do Termo de Recepção do sistema legado (AGILE).")
	p.section("Documentação de instrução da OS")
	p.docList(d)
	p.signatures([]sigSlot{
		{"Fiscal do Contrato", "Portaria de Designação", "Assinatura / Matrícula"},
		{"Ordenador de Despesa / Secretário", "Prefeitura Municipal de Rondonópolis", "Assinatura / Carimbo"},
	})
	return p.finish(d, w)
}

func renderRelatorio(d domain.MonthlyDemand, w io.Writer) error {
	p := newOfficialPDF("Relatório Mensal de Fiscalização (Anexo I — IN SCL 01/2019)", "ANEXO I", d)
	p.contractBox(d)
	p.paragraph(fmt.Sprintf(
		"Em cumprimento ao art. 117 da Lei nº 14.133/2021 e à IN SCL nº 01/2019, o fiscal do contrato "+
			"nº %s ATESTA que a empresa %s executou o objeto contratual referente à competência %s, e que "+
			"a documentação de habilitação e regularidade fiscal/trabalhista foi conferida.",
		orDash(d.ContratoNumero), orDash(d.Contratado), d.AnoMes))

	bruto := "____________"
	if d.ContratoValor != nil {
		bruto = brl(*d.ContratoValor)
	}
	p.section("Apuração de valores e retenções")
	p.table(
		[]string{"Descrição", "Valor (R$)"},
		[]float64{130, 40},
		[][]string{
			{"Valor bruto da medição / nota fiscal", bruto},
			{"(-) ISSQN", "____________"},
			{"(-) INSS", "____________"},
			{"(-) IRRF", "____________"},
			{"(-) Outras retenções", "____________"},
			{"VALOR LÍQUIDO A PAGAR", "____________"},
		},
	)

	p.section("Checklist de conformidade (Anexo I)")
	byType := map[domain.DocumentType]domain.DemandDocument{}
	for _, doc := range d.Documents {
		byType[doc.DocType] = doc
	}
	req := []domain.DocumentType{
		domain.DocExtratoEmpenho, domain.DocRelatorioPgto,
		domain.DocCertidaoFederal, domain.DocCertidaoEstadual, domain.DocCertidaoMunicipal,
		domain.DocCertidaoFGTS, domain.DocCertidaoCNDT, domain.DocCertidaoSimples,
	}
	rows := make([][]string, 0, len(req))
	now := time.Now()
	for _, dt := range req {
		doc, ok := byType[dt]
		validade, situacao := "—", "AUSENTE"
		if ok {
			situacao = "OK"
			if doc.Validade != nil {
				validade = doc.Validade.Format("02/01/2006")
				if doc.Validade.Before(now) {
					situacao = "VENCIDA"
				}
			}
		}
		rows = append(rows, []string{dt.Label(), doc.FileName, validade, situacao})
	}
	p.table([]string{"Documento", "Arquivo", "Validade", "Situação"},
		[]float64{74, 58, 20, 18}, rows)

	if strings.TrimSpace(d.Observacoes) != "" {
		p.section("Observações da Demanda")
		p.paragraph(d.Observacoes)
	}

	p.section("Carimbo eletrônico de fiscalização")
	p.paragraph(fmt.Sprintf("Contrato: %s     Competência: %s", orDash(d.ContratoNumero), d.AnoMes))
	p.paragraph("Fiscal: ____________________________   Matrícula: ______________")
	p.paragraph("Portaria de designação nº ____________   Data: ____/____/________")

	p.signatures([]sigSlot{
		{"Fiscal do Contrato", "Nome / Matrícula / Portaria", "Atesto a execução conforme o contrato"},
		{"Gestor do Contrato", "Secretaria Municipal", "Assinatura / Carimbo"},
	})
	return p.finish(d, w)
}

func (p *officialPDF) docList(d domain.MonthlyDemand) {
	pdf := p.pdf
	pdf.SetFont("Helvetica", "", 7.5)
	pdf.SetTextColor(60, 60, 60)
	if len(d.Documents) == 0 {
		pdf.MultiCell(0, 4, p.tr("Nenhum documento anexado à demanda."), "", "L", false)
		pdf.Ln(2)
		return
	}
	for _, doc := range d.Documents {
		v := ""
		if doc.Validade != nil {
			v = " (validade " + doc.Validade.Format("02/01/2006") + ")"
		}
		pdf.MultiCell(0, 4, p.tr(fmt.Sprintf("• %s — %s%s", doc.DocType.Label(), doc.FileName, v)), "", "L", false)
	}
	pdf.Ln(2)
}
