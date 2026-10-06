package repository

import (
	"sort"
	"sync"

	"example.com/paleo-prep-station/internal/domain"
)

type state struct {
	specimens         map[string]domain.Specimen
	plans             map[string]domain.PrepPlan
	progress          map[string][]domain.ProgressEntry
	releases          map[string]domain.ReleaseRecord
	residueContainers map[string]domain.ResidueContainer
	residueEntries    map[string][]domain.ResidueEntry
	residueShipments  map[string]domain.ResidueShipment
}

type MemoryRepository struct {
	mu    sync.Mutex
	state state
}

func NewMemory() *MemoryRepository {
	return &MemoryRepository{
		state: state{
			specimens:         make(map[string]domain.Specimen),
			plans:             make(map[string]domain.PrepPlan),
			progress:          make(map[string][]domain.ProgressEntry),
			releases:          make(map[string]domain.ReleaseRecord),
			residueContainers: make(map[string]domain.ResidueContainer),
			residueEntries:    make(map[string][]domain.ResidueEntry),
			residueShipments:  make(map[string]domain.ResidueShipment),
		},
	}
}

func (r *MemoryRepository) Read(fn func(*Mutation) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return fn(cloneMutation(r.state))
}

func (r *MemoryRepository) Mutate(fn func(*Mutation) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	tx := cloneMutation(r.state)
	if err := fn(tx); err != nil {
		return err
	}
	r.state = tx.snapshot()
	return nil
}

func cloneMutation(source state) *Mutation {
	copied := state{
		specimens:         make(map[string]domain.Specimen, len(source.specimens)),
		plans:             make(map[string]domain.PrepPlan, len(source.plans)),
		progress:          make(map[string][]domain.ProgressEntry, len(source.progress)),
		releases:          make(map[string]domain.ReleaseRecord, len(source.releases)),
		residueContainers: make(map[string]domain.ResidueContainer, len(source.residueContainers)),
		residueEntries:    make(map[string][]domain.ResidueEntry, len(source.residueEntries)),
		residueShipments:  make(map[string]domain.ResidueShipment, len(source.residueShipments)),
	}
	for key, value := range source.specimens {
		copied.specimens[key] = value
	}
	for key, value := range source.plans {
		copied.plans[key] = value
	}
	for key, values := range source.progress {
		copied.progress[key] = append([]domain.ProgressEntry(nil), values...)
	}
	for key, value := range source.releases {
		copied.releases[key] = value
	}
	for key, value := range source.residueContainers {
		copied.residueContainers[key] = value
	}
	for key, values := range source.residueEntries {
		copied.residueEntries[key] = append([]domain.ResidueEntry(nil), values...)
	}
	for key, value := range source.residueShipments {
		copied.residueShipments[key] = value
	}
	return &Mutation{state: copied}
}

type Mutation struct {
	state state
}

func (m *Mutation) snapshot() state {
	return m.state
}

func (m *Mutation) GetSpecimen(id string) (domain.Specimen, bool) {
	value, ok := m.state.specimens[id]
	return value, ok
}

func (m *Mutation) FindSpecimenByCatalog(catalogNumber string) (domain.Specimen, bool) {
	for _, value := range m.state.specimens {
		if value.CatalogNumber == catalogNumber {
			return value, true
		}
	}
	return domain.Specimen{}, false
}

func (m *Mutation) ListSpecimens() []domain.Specimen {
	values := make([]domain.Specimen, 0, len(m.state.specimens))
	for _, value := range m.state.specimens {
		values = append(values, value)
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].CreatedAt.Equal(values[j].CreatedAt) {
			return values[i].ID < values[j].ID
		}
		return values[i].CreatedAt.Before(values[j].CreatedAt)
	})
	return values
}

func (m *Mutation) PutSpecimen(value domain.Specimen) {
	m.state.specimens[value.ID] = value
}

func (m *Mutation) GetPlan(id string) (domain.PrepPlan, bool) {
	value, ok := m.state.plans[id]
	return value, ok
}

func (m *Mutation) ListPlansForSpecimen(specimenID string) []domain.PrepPlan {
	values := make([]domain.PrepPlan, 0)
	for _, value := range m.state.plans {
		if value.SpecimenID == specimenID {
			values = append(values, value)
		}
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].CreatedAt.Equal(values[j].CreatedAt) {
			return values[i].ID < values[j].ID
		}
		return values[i].CreatedAt.Before(values[j].CreatedAt)
	})
	return values
}

func (m *Mutation) PutPlan(value domain.PrepPlan) {
	m.state.plans[value.ID] = value
}

func (m *Mutation) HasOpenPlan(specimenID string) bool {
	for _, value := range m.state.plans {
		if value.SpecimenID == specimenID && value.IsOpen() {
			return true
		}
	}
	return false
}

func (m *Mutation) ProgressForPlan(planID string) []domain.ProgressEntry {
	return append([]domain.ProgressEntry(nil), m.state.progress[planID]...)
}

func (m *Mutation) AppendProgress(value domain.ProgressEntry) {
	m.state.progress[value.PlanID] = append(m.state.progress[value.PlanID], value)
}

func (m *Mutation) ReleaseForSpecimen(specimenID string) (domain.ReleaseRecord, bool) {
	for _, value := range m.state.releases {
		if value.SpecimenID == specimenID {
			return value, true
		}
	}
	return domain.ReleaseRecord{}, false
}

func (m *Mutation) PutRelease(value domain.ReleaseRecord) {
	m.state.releases[value.ID] = value
}

func (m *Mutation) GetResidueContainer(id string) (domain.ResidueContainer, bool) {
	value, ok := m.state.residueContainers[id]
	return value, ok
}

func (m *Mutation) FindResidueContainerByCode(code string) (domain.ResidueContainer, bool) {
	for _, value := range m.state.residueContainers {
		if value.Code == code {
			return value, true
		}
	}
	return domain.ResidueContainer{}, false
}

func (m *Mutation) PutResidueContainer(value domain.ResidueContainer) {
	m.state.residueContainers[value.ID] = value
}

func (m *Mutation) ListResidueContainers() []domain.ResidueContainer {
	values := make([]domain.ResidueContainer, 0, len(m.state.residueContainers))
	for _, value := range m.state.residueContainers {
		values = append(values, value)
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].CreatedAt.Equal(values[j].CreatedAt) {
			return values[i].ID < values[j].ID
		}
		return values[i].CreatedAt.Before(values[j].CreatedAt)
	})
	return values
}

func (m *Mutation) ResiduesForContainer(containerID string) []domain.ResidueEntry {
	return append([]domain.ResidueEntry(nil), m.state.residueEntries[containerID]...)
}

func (m *Mutation) AppendResidue(value domain.ResidueEntry) {
	m.state.residueEntries[value.ContainerID] = append(m.state.residueEntries[value.ContainerID], value)
}

func (m *Mutation) ResidueKeyExists(containerID string, entryKey string) (domain.ResidueEntry, bool) {
	for _, value := range m.state.residueEntries[containerID] {
		if value.EntryKey == entryKey {
			return value, true
		}
	}
	return domain.ResidueEntry{}, false
}

func (m *Mutation) ShipmentForContainer(containerID string) (domain.ResidueShipment, bool) {
	for _, value := range m.state.residueShipments {
		if value.ContainerID == containerID {
			return value, true
		}
	}
	return domain.ResidueShipment{}, false
}

func (m *Mutation) PutResidueShipment(value domain.ResidueShipment) {
	m.state.residueShipments[value.ID] = value
}

func (m *Mutation) ListResidueShipments() []domain.ResidueShipment {
	values := make([]domain.ResidueShipment, 0, len(m.state.residueShipments))
	for _, value := range m.state.residueShipments {
		values = append(values, value)
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].CreatedAt.Equal(values[j].CreatedAt) {
			return values[i].ID < values[j].ID
		}
		return values[i].CreatedAt.Before(values[j].CreatedAt)
	})
	return values
}
