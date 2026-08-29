package typesense

// DiorondonArticle representa a estrutura de um artigo/extrato de diário oficial no Typesense
type DiorondonArticle struct {
	ID              string   `json:"id"`
	EditionNumber   int32    `json:"edition_number"`
	EditionType     string   `json:"edition_type"`
	PublicationDate int64    `json:"publication_date"`
	PageNumber      int32    `json:"page_number"`
	ContractNumbers []string `json:"contract_numbers"`
	CNPJs           []string `json:"cnpjs"`
	OfficialsNamed  []string `json:"officials_named"`
	Content         string   `json:"content"`
	PDFStorageURL   string   `json:"pdf_storage_url,omitempty"`
}

// DiorondonPersonnelAct representa os atos funcionais (RH, Nomeações, Exonerações) no Typesense
type DiorondonPersonnelAct struct {
	ID              string  `json:"id"`
	EditionNumber   int32   `json:"edition_number"`
	EditionType     string  `json:"edition_type"`
	PublicationDate int64   `json:"publication_date"`
	ActType         string  `json:"act_type"` // NOMEACAO_EFETIVO, NOMEACAO_COMISSIONADO, CONTRATACAO_TEMPORARIA, EXONERACAO, RESCISAO, RELOTACAO, DESIGNACAO_FUNCAO
	PersonName      string  `json:"person_name"`
	PersonCPF       string  `json:"person_cpf,omitempty"`
	PersonMatricula string  `json:"person_matricula,omitempty"`
	JobRole         string  `json:"job_role"`
	Secretaria      string  `json:"secretaria"`
	DASLevel        string  `json:"das_level,omitempty"`
	SalaryValue     float64 `json:"salary_value,omitempty"`
	PortariaNumber  string  `json:"portaria_number,omitempty"`
	FullActText     string  `json:"full_act_text"`
	PDFPageNumber   int32   `json:"pdf_page_number"`
	PDFStorageURL   string  `json:"pdf_storage_url"`
	Confidence      string  `json:"confidence,omitempty"` // high | medium — a UI mostra selo em 'medium'
}
