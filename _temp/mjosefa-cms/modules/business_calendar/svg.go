//go:build !wasm

package business_calendar

import (
	"webtyp.com/svg"
	"webtyp.com/svg/sprite"
)

// Icons es un tipo que aloja IconID/IconSvg para su descubrimiento y uso en config.
// El ID se reserva ahora (para la Etapa 6, cuando exista la pantalla) — la
// convención de este repo es un solo dueño por ícono de módulo.
type Icons struct{}

func (m *Icons) IconSvg() *sprite.Sprite {
	return sprite.NewSprite(
		// Calendario simple: marco + una marca de fecha. Un solo path, currentColor.
		sprite.Define(svg.Icon(ID), "0 0 16 16",
			sprite.Path("M4 1a1 1 0 0 1 1 1v1h6V2a1 1 0 1 1 2 0v1h1a1 1 0 0 1 1 1v10a1 1 0 0 1-1 1H2a1 1 0 0 1-1-1V4a1 1 0 0 1 1-1h1V2a1 1 0 0 1 1-1zM2 6v8h12V6H2z"),
		),
	)
}
