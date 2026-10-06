package contracts

type RegisterSpecimenRequest struct {
	CatalogNumber string `json:"catalog_number"`
	FieldLabel    string `json:"field_label"`
	TaxonHint     string `json:"taxon_hint"`
	MatrixKind    string `json:"matrix_kind"`
	JacketNotes   string `json:"jacket_notes"`
}

type CreatePlanRequest struct {
	Objective      string `json:"objective"`
	Method         string `json:"method"`
	Adhesive       string `json:"adhesive"`
	TargetExposure int    `json:"target_exposure"`
}

type ApprovalRequest struct {
	ApprovedBy string `json:"approved_by"`
}

type ProgressRequest struct {
	Stage      string `json:"stage"`
	Percent    int    `json:"percent"`
	Note       string `json:"note"`
	RecordedBy string `json:"recorded_by"`
}

type ReleaseRequest struct {
	Purpose    string `json:"purpose"`
	Recipient  string `json:"recipient"`
	ApprovedBy string `json:"approved_by"`
}

type OpenResidueContainerRequest struct {
	Code          string `json:"code"`
	Kind          string `json:"kind"`
	CapacityGrams int    `json:"capacity_grams"`
}

type AddResidueRequest struct {
	EntryKey   string `json:"entry_key"`
	SpecimenID string `json:"specimen_id"`
	Grams      int    `json:"grams"`
	RecordedBy string `json:"recorded_by"`
}

type SealResidueContainerRequest struct {
	SealedBy string `json:"sealed_by"`
}
