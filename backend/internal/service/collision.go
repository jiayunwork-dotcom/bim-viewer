package service

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"bim-viewer/internal/model"
	"bim-viewer/internal/repository"
)

type CollisionService struct{}

func NewCollisionService() *CollisionService {
	return &CollisionService{}
}

type CollisionRequest struct {
	ModelID   string   `json:"modelId"`
	GroupA    []string `json:"groupA"`
	GroupB    []string `json:"groupB"`
	Threshold float64  `json:"threshold"`
}

func (cs *CollisionService) DetectCollisions(repo *repository.PostgresRepo, req CollisionRequest) ([]*model.CollisionResult, error) {
	elementsA, err := repo.GetElementsByIDs(req.GroupA)
	if err != nil {
		return nil, fmt.Errorf("failed to get group A elements: %w", err)
	}
	elementsB, err := repo.GetElementsByIDs(req.GroupB)
	if err != nil {
		return nil, fmt.Errorf("failed to get group B elements: %w", err)
	}

	if len(elementsA) == 0 || len(elementsB) == 0 {
		return nil, fmt.Errorf("one or both groups are empty")
	}

	broadPairs := cs.broadPhaseAABB(elementsA, elementsB)

	narrowResults := cs.narrowPhaseGJK(broadPairs, req.Threshold)

	return narrowResults, nil
}

type CandidatePair struct {
	ElementA *model.Element
	ElementB *model.Element
}

func (cs *CollisionService) broadPhaseAABB(groupA, groupB []*model.Element) []CandidatePair {
	var pairs []CandidatePair

	for _, a := range groupA {
		for _, b := range groupB {
			if cs.aabbIntersects(a.AABBMin, a.AABBMax, b.AABBMin, b.AABBMax) {
				pairs = append(pairs, CandidatePair{ElementA: a, ElementB: b})
			}
		}
	}

	return pairs
}

func (cs *CollisionService) aabbIntersects(minA, maxA, minB, maxB [3]float64) bool {
	for i := 0; i < 3; i++ {
		if maxA[i] < minB[i] || minA[i] > maxB[i] {
			return false
		}
	}
	return true
}

func (cs *CollisionService) aabbDistance(minA, maxA, minB, maxB [3]float64) float64 {
	var dist float64
	for i := 0; i < 3; i++ {
		if maxA[i] < minB[i] {
			d := minB[i] - maxA[i]
			dist += d * d
		} else if minA[i] > maxB[i] {
			d := minA[i] - maxB[i]
			dist += d * d
		}
	}
	return math.Sqrt(dist)
}

func (cs *CollisionService) narrowPhaseGJK(pairs []CandidatePair, threshold float64) []*model.CollisionResult {
	var results []*model.CollisionResult
	var mu sync.Mutex
	var wg sync.WaitGroup

	batchSize := 50
	for i := 0; i < len(pairs); i += batchSize {
		end := i + batchSize
		if end > len(pairs) {
			end = len(pairs)
		}
		batch := pairs[i:end]

		wg.Add(1)
		go func(b []CandidatePair) {
			defer wg.Done()
			for _, pair := range b {
				aCenter := cs.aabbCenter(pair.ElementA.AABBMin, pair.ElementA.AABBMax)
				bCenter := cs.aabbCenter(pair.ElementB.AABBMin, pair.ElementB.AABBMax)

				aExtents := cs.aabbExtents(pair.ElementA.AABBMin, pair.ElementA.AABBMax)
				bExtents := cs.aabbExtents(pair.ElementB.AABBMin, pair.ElementB.AABBMax)

				gjkResult := cs.gjkIntersect(aCenter, aExtents, bCenter, bExtents)

				if gjkResult.Intersects {
					collisionPoint := cs.computeCollisionPoint(aCenter, bCenter)
					penetration := gjkResult.Penetration

					severity := cs.classifySeverity(penetration)

					mu.Lock()
					results = append(results, &model.CollisionResult{
						ID:             generateUUID(),
						ElementAID:     pair.ElementA.ID,
						ElementAName:   pair.ElementA.Name,
						ElementAType:   pair.ElementA.Type,
						ElementAFloor:  pair.ElementA.FloorName,
						ElementBID:     pair.ElementB.ID,
						ElementBName:   pair.ElementB.Name,
						ElementBType:   pair.ElementB.Type,
						ElementBFloor:  pair.ElementB.FloorName,
						CollisionType:  "hard",
						CollisionPoint: collisionPoint,
						Penetration:    penetration,
						Severity:       severity,
					})
					mu.Unlock()
				} else {
					distance := cs.aabbDistance(
						pair.ElementA.AABBMin, pair.ElementA.AABBMax,
						pair.ElementB.AABBMin, pair.ElementB.AABBMax,
					)

					if distance < threshold && distance > 0 {
						collisionPoint := cs.computeCollisionPoint(aCenter, bCenter)
						severity := cs.classifySoftSeverity(distance, threshold)

						mu.Lock()
						results = append(results, &model.CollisionResult{
							ID:             generateUUID(),
							ElementAID:     pair.ElementA.ID,
							ElementAName:   pair.ElementA.Name,
							ElementAType:   pair.ElementA.Type,
							ElementAFloor:  pair.ElementA.FloorName,
							ElementBID:     pair.ElementB.ID,
							ElementBName:   pair.ElementB.Name,
							ElementBType:   pair.ElementB.Type,
							ElementBFloor:  pair.ElementB.FloorName,
							CollisionType:  "soft",
							CollisionPoint: collisionPoint,
							Penetration:    distance,
							Severity:       severity,
						})
						mu.Unlock()
					}
				}
			}
		}(batch)
	}

	wg.Wait()

	sort.Slice(results, func(i, j int) bool {
		order := map[string]int{"high": 0, "medium": 1, "low": 2}
		if order[results[i].Severity] != order[results[j].Severity] {
			return order[results[i].Severity] < order[results[j].Severity]
		}
		return results[i].Penetration > results[j].Penetration
	})

	return results
}

type GJKResult struct {
	Intersects  bool
	Penetration float64
	Direction   [3]float64
}

func (cs *CollisionService) gjkIntersect(centerA, extentsA, centerB, extentsB [3]float64) GJKResult {
	direction := [3]float64{
		centerB[0] - centerA[0],
		centerB[1] - centerA[1],
		centerB[2] - centerA[2],
	}

	if direction[0] == 0 && direction[1] == 0 && direction[2] == 0 {
		direction = [3]float64{1, 0, 0}
	}

	dirLen := math.Sqrt(direction[0]*direction[0] + direction[1]*direction[1] + direction[2]*direction[2])
	direction[0] /= dirLen
	direction[1] /= dirLen
	direction[2] /= dirLen

	supportA := cs.supportBox(centerA, extentsA, direction)
	supportB := cs.supportBox(centerB, extentsB, [3]float64{-direction[0], -direction[1], -direction[2]})

	simplex := [][3]float64{
		{supportA[0] - supportB[0], supportA[1] - supportB[1], supportA[2] - supportB[2]},
	}

	negDir := [3]float64{-simplex[0][0], -simplex[0][1], -simplex[0][2]}
	negDirLen := math.Sqrt(negDir[0]*negDir[0] + negDir[1]*negDir[1] + negDir[2]*negDir[2])
	if negDirLen > 0 {
		negDir[0] /= negDirLen
		negDir[1] /= negDirLen
		negDir[2] /= negDirLen
	}

	maxIter := 64
	for i := 0; i < maxIter; i++ {
		supportA = cs.supportBox(centerA, extentsA, negDir)
		supportB = cs.supportBox(centerB, extentsB, [3]float64{-negDir[0], -negDir[1], -negDir[2]})
		newPoint := [3]float64{supportA[0] - supportB[0], supportA[1] - supportB[1], supportA[2] - supportB[2]}

		dot := newPoint[0]*negDir[0] + newPoint[1]*negDir[1] + newPoint[2]*negDir[2]
		if dot < 0 {
			return GJKResult{Intersects: false}
		}

		simplex = append(simplex, newPoint)
		contains, newDir, newSimplex := cs.processSimplex(simplex)
		if contains {
			penetration := cs.computePenetration(centerA, extentsA, centerB, extentsB)
			return GJKResult{
				Intersects:  true,
				Penetration: penetration,
				Direction:   newDir,
			}
		}
		simplex = newSimplex
		negDir = newDir
	}

	return GJKResult{Intersects: false}
}

func (cs *CollisionService) supportBox(center, extents, direction [3]float64) [3]float64 {
	result := center
	if direction[0] >= 0 {
		result[0] += extents[0]
	} else {
		result[0] -= extents[0]
	}
	if direction[1] >= 0 {
		result[1] += extents[1]
	} else {
		result[1] -= extents[1]
	}
	if direction[2] >= 0 {
		result[2] += extents[2]
	} else {
		result[2] -= extents[2]
	}
	return result
}

func (cs *CollisionService) processSimplex(simplex [][3]float64) (bool, [3]float64, [][3]float64) {
	switch len(simplex) {
	case 2:
		return cs.processLineSimplex(simplex)
	case 3:
		return cs.processTriangleSimplex(simplex)
	case 4:
		return cs.processTetrahedronSimplex(simplex)
	}
	return false, [3]float64{1, 0, 0}, simplex
}

func (cs *CollisionService) processLineSimplex(simplex [][3]float64) (bool, [3]float64, [][3]float64) {
	b, a := simplex[0], simplex[1]
	ab := [3]float64{b[0] - a[0], b[1] - a[1], b[2] - a[2]}
	ao := [3]float64{-a[0], -a[1], -a[2]}

	dot := ab[0]*ao[0] + ab[1]*ao[1] + ab[2]*ao[2]
	if dot > 0 {
		cross := [3]float64{
			ab[1]*ao[2] - ab[2]*ao[1],
			ab[2]*ao[0] - ab[0]*ao[2],
			ab[0]*ao[1] - ab[1]*ao[0],
		}
		dir := [3]float64{
			cross[1]*ab[2] - cross[2]*ab[1],
			cross[2]*ab[0] - cross[0]*ab[2],
			cross[0]*ab[1] - cross[1]*ab[0],
		}
		dirLen := math.Sqrt(dir[0]*dir[0] + dir[1]*dir[1] + dir[2]*dir[2])
		if dirLen > 0 {
			dir[0] /= dirLen
			dir[1] /= dirLen
			dir[2] /= dirLen
		}
		return false, dir, simplex
	}

	dirLen := math.Sqrt(ao[0]*ao[0] + ao[1]*ao[1] + ao[2]*ao[2])
	if dirLen > 0 {
		ao[0] /= dirLen
		ao[1] /= dirLen
		ao[2] /= dirLen
	}
	return false, ao, [][3]float64{a}
}

func (cs *CollisionService) processTriangleSimplex(simplex [][3]float64) (bool, [3]float64, [][3]float64) {
	c, b, a := simplex[0], simplex[1], simplex[2]
	ab := [3]float64{b[0] - a[0], b[1] - a[1], b[2] - a[2]}
	ac := [3]float64{c[0] - a[0], c[1] - a[1], c[2] - a[2]}
	ao := [3]float64{-a[0], -a[1], -a[2]}

	abcNormal := [3]float64{
		ab[1]*ac[2] - ab[2]*ac[1],
		ab[2]*ac[0] - ab[0]*ac[2],
		ab[0]*ac[1] - ab[1]*ac[0],
	}

	dot := abcNormal[0]*ao[0] + abcNormal[1]*ao[1] + abcNormal[2]*ao[2]
	if dot > 0 {
		dirLen := math.Sqrt(abcNormal[0]*abcNormal[0] + abcNormal[1]*abcNormal[1] + abcNormal[2]*abcNormal[2])
		if dirLen > 0 {
			abcNormal[0] /= dirLen
			abcNormal[1] /= dirLen
			abcNormal[2] /= dirLen
		}
		return false, abcNormal, simplex
	}

	return false, [3]float64{-abcNormal[0], -abcNormal[1], -abcNormal[2]}, simplex
}

func (cs *CollisionService) processTetrahedronSimplex(simplex [][3]float64) (bool, [3]float64, [][3]float64) {
	d, c, b, a := simplex[0], simplex[1], simplex[2], simplex[3]
	ao := [3]float64{-a[0], -a[1], -a[2]}

	ab := [3]float64{b[0] - a[0], b[1] - a[1], b[2] - a[2]}
	ac := [3]float64{c[0] - a[0], c[1] - a[1], c[2] - a[2]}
	ad := [3]float64{d[0] - a[0], d[1] - a[1], d[2] - a[2]}

	abcNormal := [3]float64{
		ab[1]*ac[2] - ab[2]*ac[1],
		ab[2]*ac[0] - ab[0]*ac[2],
		ab[0]*ac[1] - ab[1]*ac[0],
	}
	dotABC := abcNormal[0]*ad[0] + abcNormal[1]*ad[1] + abcNormal[2]*ad[2]
	if dotABC > 0 {
		abcNormal[0] = -abcNormal[0]
		abcNormal[1] = -abcNormal[1]
		abcNormal[2] = -abcNormal[2]
	}

	acdNormal := [3]float64{
		ac[1]*ad[2] - ac[2]*ad[1],
		ac[2]*ad[0] - ac[0]*ad[2],
		ac[0]*ad[1] - ac[1]*ad[0],
	}
	dotACD := acdNormal[0]*ab[0] + acdNormal[1]*ab[1] + acdNormal[2]*ab[2]
	if dotACD > 0 {
		acdNormal[0] = -acdNormal[0]
		acdNormal[1] = -acdNormal[1]
		acdNormal[2] = -acdNormal[2]
	}

	adbNormal := [3]float64{
		ad[1]*ab[2] - ad[2]*ab[1],
		ad[2]*ab[0] - ad[0]*ab[2],
		ad[0]*ab[1] - ad[1]*ab[0],
	}
	dotADB := adbNormal[0]*ac[0] + adbNormal[1]*ac[1] + adbNormal[2]*ac[2]
	if dotADB > 0 {
		adbNormal[0] = -adbNormal[0]
		adbNormal[1] = -adbNormal[1]
		adbNormal[2] = -adbNormal[2]
	}

	dotAO_ABC := abcNormal[0]*ao[0] + abcNormal[1]*ao[1] + abcNormal[2]*ao[2]
	dotAO_ACD := acdNormal[0]*ao[0] + acdNormal[1]*ao[1] + acdNormal[2]*ao[2]
	dotAO_ADB := adbNormal[0]*ao[0] + adbNormal[1]*ao[1] + adbNormal[2]*ao[2]

	if dotAO_ABC < 0 {
		return false, abcNormal, [][3]float64{c, b, a}
	}
	if dotAO_ACD < 0 {
		return false, acdNormal, [][3]float64{d, c, a}
	}
	if dotAO_ADB < 0 {
		return false, adbNormal, [][3]float64{b, d, a}
	}

	return true, ao, simplex
}

func (cs *CollisionService) computePenetration(centerA, extentsA, centerB, extentsB [3]float64) float64 {
	overlapX := (extentsA[0] + extentsB[0]) - math.Abs(centerA[0]-centerB[0])
	overlapY := (extentsA[1] + extentsB[1]) - math.Abs(centerA[1]-centerB[1])
	overlapZ := (extentsA[2] + extentsB[2]) - math.Abs(centerA[2]-centerB[2])

	minOverlap := overlapX
	if overlapY < minOverlap {
		minOverlap = overlapY
	}
	if overlapZ < minOverlap {
		minOverlap = overlapZ
	}

	if minOverlap < 0 {
		return 0
	}
	return minOverlap
}

func (cs *CollisionService) aabbCenter(aabbMin, aabbMax [3]float64) [3]float64 {
	return [3]float64{
		(aabbMin[0] + aabbMax[0]) / 2,
		(aabbMin[1] + aabbMax[1]) / 2,
		(aabbMin[2] + aabbMax[2]) / 2,
	}
}

func (cs *CollisionService) aabbExtents(aabbMin, aabbMax [3]float64) [3]float64 {
	return [3]float64{
		(aabbMax[0] - aabbMin[0]) / 2,
		(aabbMax[1] - aabbMin[1]) / 2,
		(aabbMax[2] - aabbMin[2]) / 2,
	}
}

func (cs *CollisionService) computeCollisionPoint(centerA, centerB [3]float64) [3]float64 {
	return [3]float64{
		(centerA[0] + centerB[0]) / 2,
		(centerA[1] + centerB[1]) / 2,
		(centerA[2] + centerB[2]) / 2,
	}
}

func (cs *CollisionService) classifySeverity(penetration float64) string {
	if penetration > 100 {
		return "high"
	}
	if penetration > 30 {
		return "medium"
	}
	return "low"
}

func (cs *CollisionService) classifySoftSeverity(distance, threshold float64) string {
	ratio := distance / threshold
	if ratio < 0.3 {
		return "high"
	}
	if ratio < 0.7 {
		return "medium"
	}
	return "low"
}
