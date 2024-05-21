package bootstrap

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/healer1219/martini/cloud"
	"github.com/healer1219/martini/global"
	"github.com/healer1219/martini/mevent"
	"github.com/healer1219/martini/mlog"
	"github.com/healer1219/martini/routes"
	"log"
	"net/http"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

func init() {
	InitConfig()
	mlog.InitLog()
}

const (
	BootupEvent   = mevent.EventType("bootup")
	ShutdownEvent = mevent.EventType("shutdown")
	StartupEvent  = mevent.EventType("startup")
)

type StartFunc func(ctx *global.Context)

type StartEventHandler struct {
	StartFunc
}

func (b *StartEventHandler) OnEvent(ctx *global.Context) {
	b.StartFunc(ctx)
}

type ShutDownFunc func(ctx *global.Context)

type ShutDownEventHandler struct {
	ShutDownFunc
}

func (b *ShutDownEventHandler) OnEvent(ctx *global.Context) {
	b.ShutDownFunc(ctx)
}

type BootOption func(ctx *global.Context)

type BootEventHandler struct {
	BootOption
}

func (b *BootEventHandler) OnEvent(ctx *global.Context) {
	b.BootOption(ctx)
}

func parseEvent(fc func(ctx *global.Context), eventType mevent.EventType) mevent.Event {
	switch eventType {
	case BootupEvent:
		return &BootEventHandler{fc}
	case ShutdownEvent:
		return &ShutDownEventHandler{fc}
	case StartupEvent:
		return &StartEventHandler{fc}
	}
	return nil
}

type Bootstrap struct {
	engine          *gin.Engine
	bootOpts        []BootOption
	startOpts       []StartFunc
	shutDownOpts    []ShutDownFunc
	middleWares     []gin.HandlerFunc
	globalApp       *global.Application
	serviceInstance cloud.ServiceInstance
	registry        cloud.ServiceRegistry
}

func Default() *Bootstrap {
	app := NewApplicationWithOpts()
	return app
}

func NewApplicationWithOpts(opts ...BootOption) *Bootstrap {
	return NewApplication(
		newGin(),
		opts,
		make([]StartFunc, 0),
		global.App,
	)
}

func newGin() *gin.Engine {
	engine := gin.New()
	engine.Use(
		mlog.LoggerMiddleWare(global.Logger()),
		mlog.GinRecovery(global.Logger(), true),
	)
	return engine
}

func NewApplication(engine *gin.Engine, bootOpts []BootOption, startOpts []StartFunc, globalApp *global.Application) *Bootstrap {
	return &Bootstrap{
		engine:    engine,
		bootOpts:  bootOpts,
		startOpts: startOpts,
		globalApp: globalApp,
	}
}

func (app *Bootstrap) BootOpt(bootOpts ...BootOption) *Bootstrap {
	if app.bootOpts == nil {
		app.bootOpts = bootOpts
	} else {
		app.bootOpts = append(app.bootOpts, bootOpts...)
	}
	return app
}

func (app *Bootstrap) StartFunc(startOpts ...StartFunc) *Bootstrap {
	if app.startOpts == nil {
		app.startOpts = startOpts
	} else {
		app.startOpts = append(app.startOpts, startOpts...)
	}
	return app
}

func (app *Bootstrap) ShutDownFunc(shutDownOpts ...ShutDownFunc) *Bootstrap {
	if app.shutDownOpts == nil {
		app.shutDownOpts = shutDownOpts
	} else {
		app.shutDownOpts = append(app.shutDownOpts, shutDownOpts...)
	}
	return app
}

func (app *Bootstrap) Router(opts ...routes.RouteOption) *Bootstrap {
	routes.Register(opts...)
	return app
}

func (app *Bootstrap) Use(middleware ...gin.HandlerFunc) *Bootstrap {
	if app.middleWares == nil {
		app.middleWares = middleware
	} else {
		app.middleWares = append(app.middleWares, middleware...)
	}
	return app
}

func (app *Bootstrap) Discovery(serviceInstance cloud.ServiceInstance, registry cloud.ServiceRegistry) *Bootstrap {
	app.serviceInstance = serviceInstance
	app.registry = registry
	app.StartFunc(func(ctx *global.Context) {
		registry.Register(serviceInstance)
	})
	app.ShutDownFunc(func(ctx *global.Context) {
		registry.Deregister()
	})
	return app
}

func (app *Bootstrap) DefaultDiscovery() *Bootstrap {
	instance, err := cloud.NewDefaultServiceInstance()
	if err != nil {
		log.Fatal(err)
	}

	registry := global.Config().Cloud
	if registry.IsEmpty() {
		log.Fatal("config file [cloud] is illegal")
	}
	serviceRegistry, err := cloud.NewDefaultConsulServiceRegistry(&registry)
	if err != nil {
		log.Fatal(err)
	}

	app.Router(cloud.DefaultHealthCheck)

	return app.Discovery(instance, serviceRegistry)
}

func (app *Bootstrap) BootUp() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	for _, bootOpt := range app.bootOpts {
		event := parseEvent(bootOpt, BootupEvent)
		mevent.AddBlock(BootupEvent, event)
	}
	mevent.Publish(BootupEvent)

	for _, middleWare := range app.middleWares {
		app.engine.Use(middleWare)
	}
	routes.SetupRouter(app.engine)

	for _, startOpt := range app.startOpts {
		event := parseEvent(startOpt, StartupEvent)
		mevent.AddBlock(StartupEvent, event)
	}
	mevent.Publish(StartupEvent)

	global.App.Logger.Info("starting ------ ----- --- ")
	server := &http.Server{
		Addr:    ":" + strconv.Itoa(app.globalApp.Config.App.Port),
		Handler: app.engine,
	}

	go func() {
		err := server.ListenAndServe()
		if err != nil {
			log.Fatal("application run failed!", err)
		}
	}()

	<-ctx.Done()
	for _, shutDownOpt := range app.shutDownOpts {
		event := parseEvent(shutDownOpt, ShutdownEvent)
		mevent.AddBlock(ShutdownEvent, event)
	}
	mevent.Publish(ShutdownEvent)
	stop()
	log.Println("application is shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("application forced to shutdown: ", err)
	}

	log.Println("application exiting")
}
