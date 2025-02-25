package server

import (
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

func (s *Server) handleWebAppWS(c echo.Context) error {
	s.log.Debug("websocket handler handleWebAppWS", zap.String("channel", "webapp"))
	user, err := s.auth.GetUser(c)
	if err != nil {
		s.log.Errorw("websocket handler at handleWebAppWS", "channel", "webapp", "error", err)
		return err
	}
	err = s.sws.WebAppHandler(user.Username, c.Response(), c.Request())
	if err != nil {
		s.log.Errorw("websocket handler", "channel", "webapp", "user", user.Username, zap.Error(err))
	}
	return nil
}

func (s *Server) handlePluginWS(c echo.Context) error {
	s.log.Debug("websocket handler handlePluginWS", zap.String("channel", "plugin"))
	user, err := s.auth.GetUser(c)
	if err != nil {
		s.log.Errorw("websocket handler at handlePluginWS", "channel", "plugin", "error", err)
		return err
	}
	err = s.sws.PluginHandler(user.Username, c.Response(), c.Request())
	if err != nil {
		s.log.Errorw("websocket handler", "channel", "plugin", "user", user.Username, zap.Error(err))
	}
	return nil
}
