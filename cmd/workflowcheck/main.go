package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	"example.com/paleo-prep-station/internal/contracts"
	"example.com/paleo-prep-station/internal/httpapi"
	"example.com/paleo-prep-station/internal/repository"
	"example.com/paleo-prep-station/internal/station"
)

type specimenView struct {
	ID            string `json:"id"`
	CatalogNumber string `json:"catalog_number"`
	Status        string `json:"status"`
}

type planView struct {
	ID         string `json:"id"`
	SpecimenID string `json:"specimen_id"`
	Status     string `json:"status"`
}

type planSnapshot struct {
	Plan     planView `json:"plan"`
	Progress []struct {
		Percent int `json:"percent"`
	} `json:"progress"`
}

type releaseView struct {
	ID         string `json:"id"`
	SpecimenID string `json:"specimen_id"`
}

type residueContainerView struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	UsedGrams int    `json:"used_grams"`
}

type residueEntryView struct {
	ID       string `json:"id"`
	EntryKey string `json:"entry_key"`
	Grams    int    `json:"grams"`
}

type residueShipmentView struct {
	ID         string `json:"id"`
	TotalGrams int    `json:"total_grams"`
	EntryCount int    `json:"entry_count"`
}

type residueView struct {
	Container residueContainerView `json:"container"`
	Entries   []residueEntryView   `json:"entries"`
	Shipment  *residueShipmentView `json:"shipment"`
}

func main() {
	workflow := flag.String("workflow", "", "workflow id to check")
	flag.Parse()

	if err := run(*workflow); err != nil {
		fmt.Fprintln(os.Stderr, "workflow check failed:", err)
		os.Exit(1)
	}
	fmt.Printf("workflow check passed: %s\n", *workflow)
}

func run(workflow string) error {
	handler := httpapi.NewRouter(station.New(repository.NewMemory()))
	server := httptest.NewServer(handler)
	defer server.Close()
	client := &http.Client{Timeout: 5 * time.Second}

	switch workflow {
	case "register-specimen":
		specimen, err := registerSpecimen(client, server.URL)
		if err != nil {
			return err
		}
		if specimen.Status != "registered" {
			return fmt.Errorf("expected registered status, got %s", specimen.Status)
		}
		var fetched specimenView
		if err := requestJSON(client, http.MethodGet, server.URL+"/specimens/"+specimen.ID, nil, http.StatusOK, &fetched); err != nil {
			return err
		}
		if fetched.ID != specimen.ID {
			return errors.New("registered specimen cannot be read back")
		}
		return nil

	case "approve-prep-plan":
		specimen, err := registerSpecimen(client, server.URL)
		if err != nil {
			return err
		}
		plan, err := createPlan(client, server.URL, specimen.ID)
		if err != nil {
			return err
		}
		var approved planView
		if err := requestJSON(
			client,
			http.MethodPost,
			server.URL+"/plans/"+plan.ID+"/approval",
			contracts.ApprovalRequest{ApprovedBy: "curator-lin"},
			http.StatusOK,
			&approved,
		); err != nil {
			return err
		}
		if approved.Status != "approved" {
			return fmt.Errorf("expected approved plan, got %s", approved.Status)
		}
		var fetchedSpecimen specimenView
		if err := requestJSON(client, http.MethodGet, server.URL+"/specimens/"+specimen.ID, nil, http.StatusOK, &fetchedSpecimen); err != nil {
			return err
		}
		if fetchedSpecimen.Status != "preparing" {
			return fmt.Errorf("expected preparing specimen, got %s", fetchedSpecimen.Status)
		}
		return nil

	case "log-preparation":
		specimen, plan, err := completedPlanSetup(client, server.URL)
		if err != nil {
			return err
		}
		if err := recordProgress(client, server.URL, plan.ID, 35, "矩阵剥离"); err != nil {
			return err
		}
		if err := recordProgress(client, server.URL, plan.ID, 100, "表面加固"); err != nil {
			return err
		}
		var snapshot planSnapshot
		if err := requestJSON(client, http.MethodGet, server.URL+"/plans/"+plan.ID, nil, http.StatusOK, &snapshot); err != nil {
			return err
		}
		if snapshot.Plan.Status != "completed" || len(snapshot.Progress) != 2 {
			return errors.New("plan did not complete with exactly two progress entries")
		}
		var fetchedSpecimen specimenView
		if err := requestJSON(client, http.MethodGet, server.URL+"/specimens/"+specimen.ID, nil, http.StatusOK, &fetchedSpecimen); err != nil {
			return err
		}
		if fetchedSpecimen.Status != "stabilized" {
			return fmt.Errorf("expected stabilized specimen, got %s", fetchedSpecimen.Status)
		}
		return nil

	case "release-for-study":
		specimen, plan, err := completedPlanSetup(client, server.URL)
		if err != nil {
			return err
		}
		if err := recordProgress(client, server.URL, plan.ID, 100, "表面加固"); err != nil {
			return err
		}
		var release releaseView
		if err := requestJSON(
			client,
			http.MethodPost,
			server.URL+"/specimens/"+specimen.ID+"/release",
			contracts.ReleaseRequest{
				Purpose:    "比较解剖研究",
				Recipient:  "地层研究组",
				ApprovedBy: "keeper-zhou",
			},
			http.StatusCreated,
			&release,
		); err != nil {
			return err
		}
		if release.SpecimenID != specimen.ID {
			return errors.New("release record references the wrong specimen")
		}
		var fetched specimenView
		if err := requestJSON(client, http.MethodGet, server.URL+"/specimens/"+specimen.ID, nil, http.StatusOK, &fetched); err != nil {
			return err
		}
		if fetched.Status != "available" {
			return fmt.Errorf("expected available specimen, got %s", fetched.Status)
		}
		return nil

	case "seal-residue-container":
		specimen, err := registerSpecimen(client, server.URL)
		if err != nil {
			return err
		}
		var container residueContainerView
		if err := requestJSON(
			client,
			http.MethodPost,
			server.URL+"/residue-containers",
			contracts.OpenResidueContainerRequest{Code: "RS-2026-01", Kind: "基质粉尘", CapacityGrams: 500},
			http.StatusCreated,
			&container,
		); err != nil {
			return err
		}
		if container.Status != "open" {
			return fmt.Errorf("expected open container, got %s", container.Status)
		}
		sheet := contracts.AddResidueRequest{
			EntryKey:   "SHEET-2026-0001",
			SpecimenID: specimen.ID,
			Grams:      120,
			RecordedBy: "preparator-chen",
		}
		var entry residueEntryView
		if err := requestJSON(
			client,
			http.MethodPost,
			server.URL+"/residue-containers/"+container.ID+"/residues",
			sheet,
			http.StatusCreated,
			&entry,
		); err != nil {
			return err
		}
		var replayed residueEntryView
		if err := requestJSON(
			client,
			http.MethodPost,
			server.URL+"/residue-containers/"+container.ID+"/residues",
			sheet,
			http.StatusCreated,
			&replayed,
		); err != nil {
			return err
		}
		if replayed.ID != entry.ID {
			return errors.New("resubmitted sheet created a second residue entry")
		}
		changed := sheet
		changed.Grams = 130
		if err := requestJSON(
			client,
			http.MethodPost,
			server.URL+"/residue-containers/"+container.ID+"/residues",
			changed,
			http.StatusConflict,
			nil,
		); err != nil {
			return err
		}
		ghost := sheet
		ghost.EntryKey = "SHEET-2026-0002"
		ghost.SpecimenID = "specimen-missing"
		if err := requestJSON(
			client,
			http.MethodPost,
			server.URL+"/residue-containers/"+container.ID+"/residues",
			ghost,
			http.StatusNotFound,
			nil,
		); err != nil {
			return err
		}
		var shipment residueShipmentView
		if err := requestJSON(
			client,
			http.MethodPost,
			server.URL+"/residue-containers/"+container.ID+"/seal",
			contracts.SealResidueContainerRequest{SealedBy: "keeper-zhou"},
			http.StatusOK,
			&shipment,
		); err != nil {
			return err
		}
		if shipment.TotalGrams != 120 || shipment.EntryCount != 1 {
			return fmt.Errorf(
				"shipment fact mismatch: total=%d count=%d",
				shipment.TotalGrams,
				shipment.EntryCount,
			)
		}
		late := sheet
		late.EntryKey = "SHEET-2026-0003"
		if err := requestJSON(
			client,
			http.MethodPost,
			server.URL+"/residue-containers/"+container.ID+"/residues",
			late,
			http.StatusConflict,
			nil,
		); err != nil {
			return err
		}
		if err := requestJSON(
			client,
			http.MethodPost,
			server.URL+"/residue-containers/"+container.ID+"/seal",
			contracts.SealResidueContainerRequest{SealedBy: "keeper-wu"},
			http.StatusConflict,
			nil,
		); err != nil {
			return err
		}
		var view residueView
		if err := requestJSON(
			client,
			http.MethodGet,
			server.URL+"/residue-containers/"+container.ID,
			nil,
			http.StatusOK,
			&view,
		); err != nil {
			return err
		}
		if view.Container.Status != "sealed" || view.Container.UsedGrams != 120 {
			return fmt.Errorf(
				"container facts mismatch: status=%s used=%d",
				view.Container.Status,
				view.Container.UsedGrams,
			)
		}
		if len(view.Entries) != 1 {
			return fmt.Errorf("expected exactly one residue entry, got %d", len(view.Entries))
		}
		if view.Shipment == nil || view.Shipment.ID != shipment.ID {
			return errors.New("container must keep exactly one shipment fact")
		}
		if view.Shipment.TotalGrams != view.Container.UsedGrams ||
			view.Shipment.EntryCount != len(view.Entries) {
			return errors.New("shipment fact disagrees with container residue facts")
		}
		return nil
	default:
		return fmt.Errorf("unknown workflow %q", workflow)
	}
}

func registerSpecimen(client *http.Client, baseURL string) (specimenView, error) {
	var specimen specimenView
	err := requestJSON(
		client,
		http.MethodPost,
		baseURL+"/specimens",
		contracts.RegisterSpecimenRequest{
			CatalogNumber: "PPS-2026-001",
			FieldLabel:    "北坡-07",
			TaxonHint:     "大型脊椎动物",
			MatrixKind:    "粉砂岩",
			JacketNotes:   "夹克边角完好",
		},
		http.StatusCreated,
		&specimen,
	)
	return specimen, err
}

func createPlan(client *http.Client, baseURL, specimenID string) (planView, error) {
	var plan planView
	err := requestJSON(
		client,
		http.MethodPost,
		baseURL+"/specimens/"+specimenID+"/plans",
		contracts.CreatePlanRequest{
			Objective:      "暴露主要关节面",
			Method:         "手工显微剥离",
			Adhesive:       "可逆树脂",
			TargetExposure: 80,
		},
		http.StatusCreated,
		&plan,
	)
	return plan, err
}

func completedPlanSetup(client *http.Client, baseURL string) (specimenView, planView, error) {
	specimen, err := registerSpecimen(client, baseURL)
	if err != nil {
		return specimenView{}, planView{}, err
	}
	plan, err := createPlan(client, baseURL, specimen.ID)
	if err != nil {
		return specimenView{}, planView{}, err
	}
	var approved planView
	if err := requestJSON(
		client,
		http.MethodPost,
		baseURL+"/plans/"+plan.ID+"/approval",
		contracts.ApprovalRequest{ApprovedBy: "curator-lin"},
		http.StatusOK,
		&approved,
	); err != nil {
		return specimenView{}, planView{}, err
	}
	return specimen, approved, nil
}

func recordProgress(client *http.Client, baseURL, planID string, percent int, stage string) error {
	var ignored struct{}
	return requestJSON(
		client,
		http.MethodPost,
		baseURL+"/plans/"+planID+"/progress",
		contracts.ProgressRequest{
			Stage:      stage,
			Percent:    percent,
			Note:       "本地链路检查",
			RecordedBy: "preparator-chen",
		},
		http.StatusCreated,
		&ignored,
	)
}

func requestJSON(
	client *http.Client,
	method string,
	url string,
	body any,
	wantStatus int,
	target any,
) error {
	var payload io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return err
		}
		payload = bytes.NewReader(encoded)
	}
	request, err := http.NewRequest(method, url, payload)
	if err != nil {
		return err
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return err
	}
	if response.StatusCode != wantStatus {
		return fmt.Errorf("request %s %s returned %d: %s", method, url, response.StatusCode, string(raw))
	}
	if target == nil || len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return fmt.Errorf("decode response for %s %s: %w", method, url, err)
	}
	return nil
}
