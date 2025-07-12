package main

import (
	"cmp"
	"image/color"
	"math"
	"math/rand"
	"slices"

	"github.com/MatiasLyyra/fluid/simulation"
	rl "github.com/gen2brain/raylib-go/raylib"
)

const MaxNeighbours = 32
const ParticleCount = 4096
const Width = 768
const Height = 768
const Radius = 0.04

const NoNeighbour = math.MaxUint32

type NeighbourList [ParticleCount * MaxNeighbours]int

func HashPosition(pos, rangeLow, rangeHigh simulation.Vector2, step float32) uint32 {
	const p1 = 73856093
	const p2 = 19349663
	gridX := uint32((pos.X - rangeLow.X) / step)
	gridY := uint32((pos.Y - rangeLow.Y) / step)
	numCellsX := uint32((rangeHigh.X - rangeLow.X) / step)
	numCellsY := uint32((rangeHigh.Y - rangeLow.X) / step)
	gridX = max(0, min(gridX, numCellsX))
	gridY = max(0, min(gridY, numCellsY))
	return ((gridX * p1) ^ (gridY * p2)) % (numCellsX * numCellsY)
}

func main() {

	particles := make([]simulation.Particle, ParticleCount)

	for i := range particles {
		particles[i] = simulation.Particle{
			Position: simulation.Vector2{
				X: rand.Float32(),
				Y: rand.Float32(),
			},
		}
		particles[i].Code = HashPosition(particles[i].Position, simulation.Vector2{X: 0, Y: 0}, simulation.Vector2{X: 1, Y: 1}, 0.02)
	}

	slices.SortFunc(particles, func(a, b simulation.Particle) int { return cmp.Compare(a.Code, b.Code) })
	rl.InitWindow(Width, Height, "Fluid")
	rl.SetTargetFPS(60)

	for !rl.WindowShouldClose() {
		var maxDensity float32
		var minDensity float32 = math.MaxFloat32
		for i := range particles {
			particles[i].Density = 0
			for j := range particles {
				if i == j {
					continue
				}
				particles[i].Density += smoothingKernel(particles[i].Position.DistanceSqr(particles[j].Position), Radius)
			}
			maxDensity = max(maxDensity, particles[i].Density)
			minDensity = min(minDensity, particles[i].Density)
		}

		rl.BeginDrawing()
		rl.ClearBackground(rl.Blank)
		for _, p := range particles {
			c := max(20, uint8((p.Density-minDensity)/(maxDensity-minDensity)*255))
			rl.DrawCircle(int32(p.Position.X*Width), int32(p.Position.Y*Height), 4, color.RGBA{R: c, G: c, B: c, A: 255})
		}
		rl.DrawFPS(10, 10)
		rl.EndDrawing()
	}
}

func smoothingKernel(rSq, h float32) float32 {
	norm := math.Pi * float32(math.Pow(float64(h), 8)) / 4
	diff := max(h*h-rSq, 0)
	return norm * diff * diff * diff
}
