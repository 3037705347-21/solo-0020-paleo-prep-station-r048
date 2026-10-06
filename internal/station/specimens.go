package station

import (
	"context"

	"example.com/paleo-prep-station/internal/contracts"
	"example.com/paleo-prep-station/internal/domain"
	"example.com/paleo-prep-station/internal/repository"
)

func (s *Service) RegisterSpecimen(
	ctx context.Context,
	request contracts.RegisterSpecimenRequest,
) (domain.Specimen, error) {
	if err := ctx.Err(); err != nil {
		return domain.Specimen{}, err
	}

	var created domain.Specimen
	err := s.repo.Mutate(func(m *repository.Mutation) error {
		value, err := domain.NewSpecimen(
			s.nextID("specimen"),
			request.CatalogNumber,
			request.FieldLabel,
			request.TaxonHint,
			request.MatrixKind,
			request.JacketNotes,
			s.now(),
		)
		if err != nil {
			return err
		}
		if _, exists := m.FindSpecimenByCatalog(value.CatalogNumber); exists {
			return contracts.Conflict("duplicate_catalog_number", "目录号已存在，不能覆盖原标本")
		}
		m.PutSpecimen(value)
		created = value
		return nil
	})
	return created, err
}

type ResidueView struct {
	Container domain.ResidueContainer `json:"container"`
	Entries   []domain.ResidueEntry   `json:"entries"`
	Shipment  *domain.ResidueShipment `json:"shipment,omitempty"`
}

func (s *Service) OpenResidueContainer(
	ctx context.Context,
	request contracts.OpenResidueContainerRequest,
) (domain.ResidueContainer, error) {
	if err := ctx.Err(); err != nil {
		return domain.ResidueContainer{}, err
	}

	var created domain.ResidueContainer
	err := s.repo.Mutate(func(m *repository.Mutation) error {
		value, err := domain.NewResidueContainer(
			s.nextID("container"),
			request.Code,
			request.Kind,
			request.CapacityGrams,
			s.now(),
		)
		if err != nil {
			return err
		}
		if _, exists := m.FindResidueContainerByCode(value.Code); exists {
			return contracts.Conflict("duplicate_residue_container", "容器标识已存在，不能重复启用")
		}
		m.PutResidueContainer(value)
		created = value
		return nil
	})
	return created, err
}

func (s *Service) AddResidue(
	ctx context.Context,
	containerID string,
	request contracts.AddResidueRequest,
) (domain.ResidueEntry, error) {
	if err := ctx.Err(); err != nil {
		return domain.ResidueEntry{}, err
	}

	var created domain.ResidueEntry
	err := s.repo.Mutate(func(m *repository.Mutation) error {
		container, ok := m.GetResidueContainer(containerID)
		if !ok {
			return contracts.NotFound("residue_container_not_found", "未找到残渣容器 "+containerID)
		}
		entry, err := domain.NewResidueEntry(
			s.nextID("residue"),
			containerID,
			request.EntryKey,
			request.SpecimenID,
			request.Grams,
			request.RecordedBy,
			s.now(),
		)
		if err != nil {
			return err
		}
		if existing, exists := m.ResidueKeyExists(containerID, entry.EntryKey); exists {
			if existing.SpecimenID == entry.SpecimenID && existing.Grams == entry.Grams {
				created = existing
				return nil
			}
			return contracts.Conflict("duplicate_residue_entry", "记料单凭据已存在，同一凭据不能记录两笔不同残渣")
		}
		if err := container.AcceptResidue(entry.Grams, s.now()); err != nil {
			return err
		}
		if _, ok := m.GetSpecimen(entry.SpecimenID); !ok {
			return notFoundSpecimen(entry.SpecimenID)
		}
		m.AppendResidue(entry)
		m.PutResidueContainer(container)
		created = entry
		return nil
	})
	return created, err
}

func (s *Service) SealResidueContainer(
	ctx context.Context,
	containerID string,
	request contracts.SealResidueContainerRequest,
) (domain.ResidueShipment, error) {
	if err := ctx.Err(); err != nil {
		return domain.ResidueShipment{}, err
	}

	var created domain.ResidueShipment
	err := s.repo.Mutate(func(m *repository.Mutation) error {
		container, ok := m.GetResidueContainer(containerID)
		if !ok {
			return contracts.NotFound("residue_container_not_found", "未找到残渣容器 "+containerID)
		}
		if _, exists := m.ShipmentForContainer(containerID); exists {
			return contracts.Conflict("residue_shipment_exists", "该容器已留下交运事实，不能重复封存")
		}
		entries := m.ResiduesForContainer(containerID)
		shipment, err := domain.NewResidueShipment(
			s.nextID("shipment"),
			container,
			request.SealedBy,
			entries,
			s.now(),
		)
		if err != nil {
			return err
		}
		if err := container.Seal(shipment.SealedBy, s.now()); err != nil {
			return err
		}
		m.PutResidueContainer(container)
		m.PutResidueShipment(shipment)
		created = shipment
		return nil
	})
	return created, err
}

func (s *Service) GetResidueContainer(ctx context.Context, containerID string) (ResidueView, error) {
	if err := ctx.Err(); err != nil {
		return ResidueView{}, err
	}
	var view ResidueView
	err := s.repo.Read(func(m *repository.Mutation) error {
		container, ok := m.GetResidueContainer(containerID)
		if !ok {
			return contracts.NotFound("residue_container_not_found", "未找到残渣容器 "+containerID)
		}
		view = ResidueView{
			Container: container,
			Entries:   m.ResiduesForContainer(containerID),
		}
		if shipment, ok := m.ShipmentForContainer(containerID); ok {
			view.Shipment = &shipment
		}
		return nil
	})
	return view, err
}
