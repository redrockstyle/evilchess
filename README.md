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
* WASM Test-Build (it works LOL)
* My AI Engine (not integrated)

## Functionallity (in the future)
* Chess Model Intagration

---

<h2 align="center">GUI Preview</h2>
<p align="center">
  <img src="materials/img/demo.gif" alt="demo" width="600">
</p>

---

<h1 align="center">Comparison: AI ​​Model vs Internal Engine</h1>

### Summary
I implemented a small [internal chess](#internal-engine-overview) engine that performs exhaustive depth-first move search and relies largely on a material-based evaluation. I trained [AI model](#chess-ai-model-overview) (as far as my resources allowed) to predict moves from FEN -> Move snapshots extracted from a filtered subset of [Lichess games](https://database.lichess.org/) - sep 2025 dataset. Training ran for **46 epochs** and lasted about **10 hours** until EarlyStopping triggered (overfitting trigger).

<p align="center">
  <img src="materials/img/training_history.png" alt="train" width="600">
</p>

Unfortunately the model was not integrated into the app, but you can download and test it locally from the [repository directory](/ai/learn/).

### Comparative tests
I ran several head-to-head and task-based comparisons between the internal engine and the neural model:

1. **Tactical puzzles (small composed tasks)**: Amateur players will quickly master these tasks - this is also reproduced by the internal engine, whereas the AI ​​model was unable to do this reliably.
<p align="center">
  <img src="materials/versus/board/total.png" alt="total" width="800">
</p>

2. [**Five personal matches(PGN)**](materials/versus/chess_ai_tour.pgn): AI model 4 - Internal engine 1 (clear superiority of the AI in these encounters).

### Observations / Qualitative behavior
- The AI model learned several positional principles: piece development toward the center, space-gaining plans, and even material sacrifices to increase activity.  
- The model showed a strong tendency to execute horizontal/linear back-rank mates when possible.  
- The network is relatively small, and its combinational/calculation depth is limited - it struggles with long tactical variations. This weakness led to at least one game where the internal engine delivered an early mate via a simple combination.

Overall: the AI emphasizes positional play; the internal engine remains superior on short tactical puzzles and precise combination calculation.

---

<h2 align="center">Chess AI Model Overview</h2>

```mermaid
flowchart TD

A["Input: Board(13×8×8) and Rating(1)"] --> B["Stem: Conv7×7 → BN → ReLU"]
B --> C[Residual Blocks: 96->128->128->192->192]
C --> X{Use Transformer?}
X -->|Yes| D["Flatten to 64x192 + Positional Embedding"]
D --> E["TinyTransformer: 2 layers, 8 heads"]
E --> F["Mean Pool -> Feature (192)"]
X -->|No| G[Global Avg Pool: 8×8 → 1×1]
G --> H["Flatten -> Feature (192)"]
F --> I[Concat Rating]
H --> I[Concat Rating]
I --> J["FC Common: (192+1->512) ReLU + Dropout"]
J --> K1[From-Head: 512->64]
J --> K2[To-Head: 512->64]
J --> K3[Value-Head: 512->64->1]
K1:::head
K2:::head
K3:::head
A:::head
classDef head fill:#0000f4,stroke:#333,stroke-width:1px,font-size:12px;
```

---

<h2 align="center">Internal Engine Overview</h2>

```mermaid
flowchart TD
  A[Start Analysis]:::theme
  A --> B[Depth = 1]:::theme
  B --> E[Generate Moves]:::theme
  E --> F[Alpha-Beta Search + TT]:::theme
  F --> G{Quiescence at Leaf}:::theme
  G -->|Yes| H[Capture-Only Search]:::theme
  G -->|No| I[TT Probe/Eval]:::theme
  H --> I
  I --> J[Publish Analysis]:::theme
  J --> L{Depth < Max}:::theme
  L -->|Yes| E
  L -->|No| M[Stop Analysis]:::theme

  classDef theme font-size:12px
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
- [Lichess Open Database](https://database.lichess.org/)
