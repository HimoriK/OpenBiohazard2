package render

import (
	"fmt"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/OpenBiohazard2/OpenBiohazard2/fileio"
)

const (
	RENDER_GAME_STATE_MAIN                   = 0
	RENDER_GAME_STATE_BACKGROUND_SOLID       = 1
	RENDER_GAME_STATE_BACKGROUND_TRANSPARENT = 2
	RENDER_TYPE_ITEM                         = 5
)

type RenderDef struct {
	ProgramShader    uint32
	Camera           *Camera
	ProjectionMatrix mgl32.Mat4
	ViewMatrix       mgl32.Mat4
	EnvironmentLight [3]float32
	WindowWidth      int
	WindowHeight     int
	VideoBuffer      *Surface2D

	SpriteGroupEntity     *SpriteGroupEntity
	BackgroundImageEntity *SceneEntity
	CameraMaskEntity      *SceneEntity
	ItemGroupEntity       *ItemGroupEntity
}

type DebugEntities struct {
	CameraSwitchDebugEntity *DebugEntity
	DebugEntities           []*DebugEntity
}

type RenderRoom struct {
	CameraMaskData  [][]fileio.MaskRectangle
	LightData       []fileio.LITCameraLight
	ItemTextureData []*fileio.TIMOutput
	ItemModelData   []*fileio.MD1Output
	SpriteData      []fileio.SpriteData
}

func InitRenderer(windowWidth int, windowHeight int) *RenderDef {
	if err := gl.Init(); err != nil {
		panic(err)
	}

	version := gl.GoStr(gl.GetString(gl.VERSION))
	fmt.Println("OpenGL version", version)

	shader := NewShader("render/openbiohazard2.vert", "render/openbiohazard2.frag")

	gl.ClearColor(0.0, 0.0, 0.0, 1.0)
	gl.Enable(gl.DEPTH_TEST)
	gl.Enable(gl.MULTISAMPLE)
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)

	cameraUp := mgl32.Vec3{0, -1, 0}
	camera := NewCamera(mgl32.Vec3{0, 0, 0}, mgl32.Vec3{0, 0, 0}, cameraUp, 60.0)
	renderDef := &RenderDef{
		ProgramShader:    shader.ProgramShader,
		Camera:           camera,
		ProjectionMatrix: mgl32.Perspective(mgl32.DegToRad(60.0), float32(4/3), 16, 45000),
		ViewMatrix:       mgl32.Ident4(),
		WindowWidth:      windowWidth,
		WindowHeight:     windowHeight,
		VideoBuffer:      NewSurface2D(),

		BackgroundImageEntity: NewBackgroundImageEntity(),
		CameraMaskEntity:      NewSceneEntity(),
		ItemGroupEntity:       NewItemGroupEntity(),
	}
	return renderDef
}

func (r *RenderDef) RenderFrame(playerEntity PlayerEntity,
	debugEntities DebugEntities,
	timeElapsedSeconds float64) {

	programShader := r.ProgramShader

	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

	// Activate shader
	gl.UseProgram(programShader)

	renderGameStateUniform := gl.GetUniformLocation(programShader, gl.Str("gameState\x00"))
	gl.Uniform1i(renderGameStateUniform, RENDER_GAME_STATE_MAIN)

	viewLoc := gl.GetUniformLocation(programShader, gl.Str("view\x00"))
	projectionLoc := gl.GetUniformLocation(programShader, gl.Str("projection\x00"))

	r.ProjectionMatrix = r.GetPerspectiveMatrix(r.Camera.CameraFov)

	// Pass the matrices to the shader
	gl.UniformMatrix4fv(viewLoc, 1, false, &r.ViewMatrix[0])
	gl.UniformMatrix4fv(projectionLoc, 1, false, &r.ProjectionMatrix[0])

	r.RenderBackground()
	for _, itemEntity := range r.ItemGroupEntity.ModelObjectData {
		r.RenderStaticEntity(*itemEntity, RENDER_TYPE_ITEM)
	}

	envLightLoc := gl.GetUniformLocation(r.ProgramShader, gl.Str("envLight\x00"))
	gl.Uniform3fv(envLightLoc, 1, &r.EnvironmentLight[0])
	RenderAnimatedEntity(programShader, playerEntity, timeElapsedSeconds)

	// RenderSprites(programShader, r.SpriteGroupEntity, timeElapsedSeconds)

	// Only render for debugging
	RenderCameraSwitches(programShader, debugEntities.CameraSwitchDebugEntity)
	RenderDebugEntities(programShader, debugEntities.DebugEntities)
}

func (r *RenderDef) RenderBackground() {
	r.RenderSceneEntity(r.BackgroundImageEntity, RENDER_GAME_STATE_BACKGROUND_SOLID)
	r.RenderSceneEntity(r.CameraMaskEntity, RENDER_GAME_STATE_BACKGROUND_TRANSPARENT)
}

func NewRenderRoom(rdtOutput *fileio.RDTOutput) RenderRoom {
	return RenderRoom{
		CameraMaskData:  rdtOutput.RIDOutput.CameraMasks,
		LightData:       rdtOutput.LightData.Lights,
		ItemTextureData: rdtOutput.ItemTextureData,
		ItemModelData:   rdtOutput.ItemModelData,
		SpriteData:      rdtOutput.SpriteOutput.SpriteData,
	}
}

func BuildEnvironmentLight(light fileio.LITCameraLight) [3]float32 {
	lightColor := light.AmbientColor
	// Normalize color rgb values to be between 0.0 and 1.0
	red := float32(lightColor.R) / float32(255.0)
	green := float32(lightColor.G) / float32(255.0)
	blue := float32(lightColor.B) / float32(255.0)
	return [3]float32{red, green, blue}
}

func (r *RenderDef) GetPerspectiveMatrix(fovDegrees float32) mgl32.Mat4 {
	ratio := float64(r.WindowWidth) / float64(r.WindowHeight)
	return mgl32.Perspective(mgl32.DegToRad(fovDegrees), float32(ratio), 16, 45000)
}
