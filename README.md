<h1 align="center">EvilChess Project</h1>

## Functionallity (for now):
* Rules of the game
* History moves
* Color CLI
* R/W FEN (Forsyth–Edwards Notation)
* R/W PGN (Portable Game Notation)
* R/W SAN (Standard Algebraic Notation)
* Internal Engine (Iterative Deep Search)
* Engine support via UCI protocol
* Color GUI (Ebiten Library):
* * Play Scene
* * Editor Scene
* * Analyzer Scene
* * Settings Scene
* WASM Test-Build (it's work LOL)

## Functionallity (in the future)
* My AI Engine (experimental)

---

<h2 align="center">GUI Preview</h2>
<p align="center">
  <img src="materials/img/demo.gif" alt="demo" width="600">
</p>

---

<h2 align="center">Internal Engine Overview</h2>

```mermaid
flowchart TD
  A[Set Position via FEN]:::theme --> B[Start Analysis]:::theme
  B --> C{Iterative Deepening}:::theme
  C --> D[Depth = 1]:::theme
  D --> E[Generate Root Moves]:::theme
  E --> F[Alpha-Beta Search + TT]:::theme
  F --> G{Quiescence at Leaf?}:::theme
  G -->|Yes| H[Capture-Only Search]:::theme
  G -->|No| I[TT Probe/Eval]:::theme
  H --> I
  I --> J[Update Best PV]:::theme
  J --> K[Publish Analysis]:::theme
  K --> L{Depth < Max?}:::theme
  L -->|Yes| D
  L -->|No| M[Stop Analysis]:::theme

  classDef theme font-size:10px
```
---

## References

- [PGN Wiki](https://en.wikipedia.org/wiki/Portable_Game_Notation)
- [FEN Wiki](https://en.wikipedia.org/wiki/Forsyth%E2%80%93Edwards_Notation)
- [Ebiten Docs](https://ebitengine.org/en/documents/)
- [Piece Images](https://commons.wikimedia.org/wiki/Category:PNG_chess_pieces/Standard_transparent)
- [Crown Image](https://www.pngwing.com/en/free-png-ntlel)
- [Font NotoSansDisplay](https://fonts.google.com/noto/specimen/Noto+Sans+Display)
- [Font PressStart2P](https://fonts.google.com/specimen/Press+Start+2P)
- [Stockfish Engine](https://stockfishchess.org/download/)
