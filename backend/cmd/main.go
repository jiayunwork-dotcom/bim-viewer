package main

import (
	"log"
	"net/http"
	"os"
	"bim-viewer/internal/handler"
	"bim-viewer/internal/repository"
	"bim-viewer/internal/service"
	"bim-viewer/internal/middleware"
	"github.com/gorilla/mux"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://bimuser:bimpanass@localhost:5432/bimviewer?sslmode=disable"
	}

	repo, err := repository.NewPostgresRepo(dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer repo.Close()

	if err := repo.Migrate(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	if err := repo.MigrateAnnotations(); err != nil {
		log.Fatalf("Failed to run annotation migrations: %v", err)
	}
	if err := repo.MigrateConstruction(); err != nil {
		log.Fatalf("Failed to run construction migrations: %v", err)
	}

	ifcParser := service.NewIFCParserService()
	collisionSvc := service.NewCollisionService()
	modelSvc := service.NewModelService(repo, ifcParser)

	wsHub := service.NewWSHub()
	go wsHub.Run()

	annotationSvc := service.NewAnnotationService(repo, wsHub)
	constructionSvc := service.NewConstructionService(repo)

	modelHandler := handler.NewModelHandler(modelSvc)
	collisionHandler := handler.NewCollisionHandler(collisionSvc, modelSvc)
	annotationHandler := handler.NewAnnotationHandler(annotationSvc, wsHub)
	constructionHandler := handler.NewConstructionHandler(constructionSvc)
	versionHandler := handler.NewVersionHandler(modelSvc)

	r := mux.NewRouter()
	api := r.PathPrefix("/api/v1").Subrouter()
	api.Use(middleware.CORS)

	api.HandleFunc("/models", modelHandler.UploadModel).Methods("POST", "OPTIONS")
	api.HandleFunc("/models", modelHandler.ListModels).Methods("GET", "OPTIONS")
	api.HandleFunc("/models/{id}", modelHandler.GetModel).Methods("GET", "OPTIONS")
	api.HandleFunc("/models/{id}", modelHandler.DeleteModel).Methods("DELETE", "OPTIONS")
	api.HandleFunc("/models/{id}/tree", modelHandler.GetSpatialTree).Methods("GET", "OPTIONS")
	api.HandleFunc("/models/{id}/elements/{elementId}", modelHandler.GetElement).Methods("GET", "OPTIONS")
	api.HandleFunc("/models/{id}/elements", modelHandler.GetElementsByType).Methods("GET", "OPTIONS")
	api.HandleFunc("/models/{id}/meshes/{lod}", modelHandler.GetMeshChunk).Methods("GET", "OPTIONS")
	api.HandleFunc("/models/{id}/octree", modelHandler.GetOctree).Methods("GET", "OPTIONS")

	api.HandleFunc("/collision/detect", collisionHandler.DetectCollisions).Methods("POST", "OPTIONS")
	api.HandleFunc("/collision/results/{taskId}", collisionHandler.GetCollisionResults).Methods("GET", "OPTIONS")
	api.HandleFunc("/collision/export/{taskId}", collisionHandler.ExportCSV).Methods("GET", "OPTIONS")
	api.HandleFunc("/collision/stats/{taskId}", collisionHandler.GetCollisionStats).Methods("GET", "OPTIONS")
	api.HandleFunc("/collision/model/{modelId}/stats", collisionHandler.GetCollisionStatsByModel).Methods("GET", "OPTIONS")
	api.HandleFunc("/collision/model/{modelId}/results", collisionHandler.GetCollisionResultsByModel).Methods("GET", "OPTIONS")
	api.HandleFunc("/collision/model/{modelId}/tasks", collisionHandler.GetCollisionTasksByModel).Methods("GET", "OPTIONS")
	api.HandleFunc("/collision/history/{resultId}", collisionHandler.GetCollisionHistory).Methods("GET", "OPTIONS")
	api.HandleFunc("/collision/results/{resultId}/status", collisionHandler.UpdateCollisionStatus).Methods("PUT", "OPTIONS")
	api.HandleFunc("/collision/results/batch/status", collisionHandler.BatchUpdateCollisionStatus).Methods("PUT", "OPTIONS")

	api.HandleFunc("/annotations", annotationHandler.CreateAnnotation).Methods("POST", "OPTIONS")
	api.HandleFunc("/annotations", annotationHandler.ListAnnotations).Methods("GET", "OPTIONS")
	api.HandleFunc("/annotations/{id}", annotationHandler.GetAnnotation).Methods("GET", "OPTIONS")
	api.HandleFunc("/annotations/{id}", annotationHandler.UpdateAnnotation).Methods("PUT", "OPTIONS")
	api.HandleFunc("/annotations/{id}", annotationHandler.DeleteAnnotation).Methods("DELETE", "OPTIONS")
	api.HandleFunc("/annotations/{id}/comments", annotationHandler.AddComment).Methods("POST", "OPTIONS")
	api.HandleFunc("/annotations/{id}/comments", annotationHandler.GetComments).Methods("GET", "OPTIONS")
	api.HandleFunc("/annotations/comments/{commentId}", annotationHandler.DeleteComment).Methods("DELETE", "OPTIONS")
	api.HandleFunc("/annotations/attachments/{filename}", annotationHandler.GetAttachment).Methods("GET", "OPTIONS")
	api.HandleFunc("/annotations/sync", annotationHandler.GetAnnotationsSince).Methods("GET", "OPTIONS")

	api.HandleFunc("/issues", annotationHandler.CreateIssue).Methods("POST", "OPTIONS")
	api.HandleFunc("/issues", annotationHandler.ListIssues).Methods("GET", "OPTIONS")
	api.HandleFunc("/issues/due-soon", annotationHandler.GetIssuesDueSoon).Methods("GET", "OPTIONS")
	api.HandleFunc("/issues/{id}", annotationHandler.GetIssue).Methods("GET", "OPTIONS")
	api.HandleFunc("/issues/{id}", annotationHandler.UpdateIssue).Methods("PUT", "OPTIONS")
	api.HandleFunc("/issues/{id}/archive", annotationHandler.ArchiveIssue).Methods("POST", "OPTIONS")

	api.HandleFunc("/construction/plans", constructionHandler.CreatePlan).Methods("POST", "OPTIONS")
	api.HandleFunc("/construction/plans", constructionHandler.ListPlans).Methods("GET", "OPTIONS")
	api.HandleFunc("/construction/plans/{id}", constructionHandler.GetPlan).Methods("GET", "OPTIONS")
	api.HandleFunc("/construction/plans/{id}", constructionHandler.UpdatePlan).Methods("PUT", "OPTIONS")
	api.HandleFunc("/construction/plans/{id}", constructionHandler.DeletePlan).Methods("DELETE", "OPTIONS")
	api.HandleFunc("/construction/plans/{planId}/phases", constructionHandler.CreatePhase).Methods("POST", "OPTIONS")
	api.HandleFunc("/construction/plans/{planId}/phases", constructionHandler.ListPhases).Methods("GET", "OPTIONS")
	api.HandleFunc("/construction/plans/{planId}/phases/{phaseId}", constructionHandler.GetPhase).Methods("GET", "OPTIONS")
	api.HandleFunc("/construction/plans/{planId}/phases/{phaseId}", constructionHandler.UpdatePhase).Methods("PUT", "OPTIONS")
	api.HandleFunc("/construction/plans/{planId}/phases/{phaseId}", constructionHandler.DeletePhase).Methods("DELETE", "OPTIONS")
	api.HandleFunc("/construction/plans/{planId}/critical-path", constructionHandler.GetCriticalPath).Methods("GET", "OPTIONS")

	api.HandleFunc("/models/{modelId}/versions", versionHandler.CreateVersion).Methods("POST", "OPTIONS")
	api.HandleFunc("/models/{modelId}/versions", versionHandler.ListVersions).Methods("GET", "OPTIONS")
	api.HandleFunc("/models/{modelId}/versions/compare", versionHandler.CompareVersions).Methods("POST", "OPTIONS")
	api.HandleFunc("/versions/{versionId}", versionHandler.GetVersion).Methods("GET", "OPTIONS")
	api.HandleFunc("/versions/{versionId}", versionHandler.DeleteVersion).Methods("DELETE", "OPTIONS")
	api.HandleFunc("/versions/{versionId}/elements/{elementId}", versionHandler.GetVersionElement).Methods("GET", "OPTIONS")

	r.HandleFunc("/ws/annotations", annotationHandler.HandleWebSocket)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("BIM Viewer API server starting on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
