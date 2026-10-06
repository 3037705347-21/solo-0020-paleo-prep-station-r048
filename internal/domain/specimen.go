package domain

import (
	"strings"
	"time"

	"example.com/paleo-prep-station/internal/contracts"
)

type SpecimenStatus string

const (
	SpecimenRegistered SpecimenStatus = "registered"
	SpecimenPlanned    SpecimenStatus = "planned"
	SpecimenPreparing  SpecimenStatus = "preparing"
	SpecimenStabilized SpecimenStatus = "stabilized"
	SpecimenAvailable  SpecimenStatus = "available"
)

type Specimen struct {
	ID            string         `json:"id"`
	CatalogNumber string         `json:"catalog_number"`
	FieldLabel    string         `json:"field_label"`
	TaxonHint     string         `json:"taxon_hint"`
	MatrixKind    string         `json:"matrix_kind"`
	JacketNotes   string         `json:"jacket_notes"`
	Status        SpecimenStatus `json:"status"`
	Version       int            `json:"version"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

func NewSpecimen(
	id string,
	catalogNumber string,
	fieldLabel string,
	taxonHint string,
	matrixKind string,
	jacketNotes string,
	now time.Time,
) (Specimen, error) {
	if id == "" {
		return Specimen{}, contracts.Internal("missing_generated_id", "标本标识未生成", nil)
	}
	catalog, err := validateCatalogNumber(catalogNumber)
	if err != nil {
		return Specimen{}, err
	}
	field, err := requiredText("野外标签", fieldLabel, 80)
	if err != nil {
		return Specimen{}, err
	}
	taxon, err := optionalText("分类提示", taxonHint, 120)
	if err != nil {
		return Specimen{}, err
	}
	matrix, err := requiredText("基质类型", matrixKind, 60)
	if err != nil {
		return Specimen{}, err
	}
	notes, err := optionalText("夹克备注", jacketNotes, 500)
	if err != nil {
		return Specimen{}, err
	}
	return Specimen{
		ID:            id,
		CatalogNumber: catalog,
		FieldLabel:    field,
		TaxonHint:     taxon,
		MatrixKind:    matrix,
		JacketNotes:   notes,
		Status:        SpecimenRegistered,
		Version:       1,
		CreatedAt:     now.UTC(),
		UpdatedAt:     now.UTC(),
	}, nil
}

func (s Specimen) CanCreatePlan() bool {
	return s.Status == SpecimenRegistered || s.Status == SpecimenPlanned
}

func (s Specimen) IsStable() bool {
	return s.Status == SpecimenStabilized
}

func (s *Specimen) MarkPlanned(now time.Time) error {
	if s.Status == SpecimenPlanned {
		return nil
	}
	if s.Status != SpecimenRegistered {
		return contracts.Conflict("specimen_not_plannable", "当前标本状态不能创建修复方案")
	}
	s.Status = SpecimenPlanned
	s.Version++
	s.UpdatedAt = now.UTC()
	return nil
}

func (s *Specimen) MarkPreparing(now time.Time) error {
	if s.Status == SpecimenPreparing {
		return nil
	}
	if s.Status != SpecimenPlanned {
		return contracts.Conflict("specimen_not_preparing", "只有已规划标本才能进入制备状态")
	}
	s.Status = SpecimenPreparing
	s.Version++
	s.UpdatedAt = now.UTC()
	return nil
}

func (s *Specimen) MarkStabilized(now time.Time) error {
	if s.Status == SpecimenStabilized {
		return nil
	}
	if s.Status != SpecimenPreparing {
		return contracts.Conflict("specimen_not_stabilizable", "只有正在制备的标本才能进入稳定状态")
	}
	s.Status = SpecimenStabilized
	s.Version++
	s.UpdatedAt = now.UTC()
	return nil
}

func (s *Specimen) MarkAvailable(now time.Time) error {
	if s.Status != SpecimenStabilized {
		return contracts.Conflict("specimen_not_releasable", "只有已稳定标本可以办理放行")
	}
	s.Status = SpecimenAvailable
	s.Version++
	s.UpdatedAt = now.UTC()
	return nil
}

type ResidueContainerStatus string

const (
	ResidueContainerOpen   ResidueContainerStatus = "open"
	ResidueContainerSealed ResidueContainerStatus = "sealed"
)

type ResidueEntry struct {
	ID          string    `json:"id"`
	EntryKey    string    `json:"entry_key"`
	ContainerID string    `json:"container_id"`
	SpecimenID  string    `json:"specimen_id"`
	Grams       int       `json:"grams"`
	RecordedBy  string    `json:"recorded_by"`
	CreatedAt   time.Time `json:"created_at"`
}

type ResidueShipment struct {
	ID          string         `json:"id"`
	ContainerID string         `json:"container_id"`
	SealedBy    string         `json:"sealed_by"`
	TotalGrams  int            `json:"total_grams"`
	EntryCount  int            `json:"entry_count"`
	Entries     []ResidueEntry `json:"entries"`
	CreatedAt   time.Time      `json:"created_at"`
}

type ResidueContainer struct {
	ID            string                 `json:"id"`
	Code          string                 `json:"code"`
	Kind          string                 `json:"kind"`
	CapacityGrams int                    `json:"capacity_grams"`
	UsedGrams     int                    `json:"used_grams"`
	Status        ResidueContainerStatus `json:"status"`
	SealedBy      string                 `json:"sealed_by,omitempty"`
	SealedAt      *time.Time             `json:"sealed_at,omitempty"`
	Version       int                    `json:"version"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

func NewResidueContainer(
	id string,
	code string,
	kind string,
	capacityGrams int,
	now time.Time,
) (ResidueContainer, error) {
	if id == "" {
		return ResidueContainer{}, contracts.Internal("missing_generated_id", "残渣容器标识未生成", nil)
	}
	codeValue, err := requiredText("容器标识", code, 40)
	if err != nil {
		return ResidueContainer{}, err
	}
	kindValue, err := requiredText("残渣种类", kind, 40)
	if err != nil {
		return ResidueContainer{}, err
	}
	if capacityGrams <= 0 {
		return ResidueContainer{}, contracts.Invalid("invalid_capacity", "容器装填上限必须大于零")
	}
	return ResidueContainer{
		ID:            id,
		Code:          codeValue,
		Kind:          kindValue,
		CapacityGrams: capacityGrams,
		Status:        ResidueContainerOpen,
		Version:       1,
		CreatedAt:     now.UTC(),
		UpdatedAt:     now.UTC(),
	}, nil
}

func (c ResidueContainer) IsOpen() bool {
	return c.Status == ResidueContainerOpen
}

func (c *ResidueContainer) AcceptResidue(grams int, now time.Time) error {
	if !c.IsOpen() {
		return contracts.Conflict("residue_container_sealed", "残渣容器已封存，不能再补记残渣")
	}
	if grams <= 0 {
		return contracts.Invalid("invalid_residue_weight", "残渣重量必须大于零")
	}
	if c.UsedGrams+grams > c.CapacityGrams {
		return contracts.Conflict("residue_container_full", "残渣容器装填量超过上限")
	}
	c.UsedGrams += grams
	c.Version++
	c.UpdatedAt = now.UTC()
	return nil
}

func (c *ResidueContainer) Seal(sealedBy string, now time.Time) error {
	if c.Status == ResidueContainerSealed {
		return contracts.Conflict("residue_container_sealed", "残渣容器已封存，不能重复办理交运")
	}
	sealer, err := requiredText("封存人", sealedBy, 80)
	if err != nil {
		return err
	}
	stamp := now.UTC()
	c.Status = ResidueContainerSealed
	c.SealedBy = sealer
	c.SealedAt = &stamp
	c.Version++
	c.UpdatedAt = stamp
	return nil
}

func NewResidueEntry(
	id string,
	containerID string,
	entryKey string,
	specimenID string,
	grams int,
	recordedBy string,
	now time.Time,
) (ResidueEntry, error) {
	if id == "" {
		return ResidueEntry{}, contracts.Internal("missing_generated_id", "残渣记录标识未生成", nil)
	}
	if containerID == "" {
		return ResidueEntry{}, contracts.Internal("missing_container_id", "残渣容器未指定", nil)
	}
	key, err := requiredText("记料单凭据", entryKey, 80)
	if err != nil {
		return ResidueEntry{}, err
	}
	if strings.TrimSpace(specimenID) == "" {
		return ResidueEntry{}, contracts.Invalid("missing_specimen", "残渣必须挂到真实存在的标本上")
	}
	if grams <= 0 {
		return ResidueEntry{}, contracts.Invalid("invalid_residue_weight", "残渣重量必须大于零")
	}
	recorder, err := requiredText("记录人", recordedBy, 80)
	if err != nil {
		return ResidueEntry{}, err
	}
	return ResidueEntry{
		ID:          id,
		EntryKey:    key,
		ContainerID: containerID,
		SpecimenID:  specimenID,
		Grams:       grams,
		RecordedBy:  recorder,
		CreatedAt:   now.UTC(),
	}, nil
}

func NewResidueShipment(
	id string,
	container ResidueContainer,
	sealedBy string,
	entries []ResidueEntry,
	now time.Time,
) (ResidueShipment, error) {
	if id == "" {
		return ResidueShipment{}, contracts.Internal("missing_generated_id", "交运事实标识未生成", nil)
	}
	if container.ID == "" {
		return ResidueShipment{}, contracts.Internal("missing_container_id", "残渣容器未指定", nil)
	}
	sealer, err := requiredText("封存人", sealedBy, 80)
	if err != nil {
		return ResidueShipment{}, err
	}
	total := 0
	frozen := make([]ResidueEntry, 0, len(entries))
	for _, entry := range entries {
		total += entry.Grams
		frozen = append(frozen, entry)
	}
	return ResidueShipment{
		ID:          id,
		ContainerID: container.ID,
		SealedBy:    sealer,
		TotalGrams:  total,
		EntryCount:  len(frozen),
		Entries:     frozen,
		CreatedAt:   now.UTC(),
	}, nil
}
