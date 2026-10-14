# Estimaciones de Issues

Análisis de las [issues abiertas](https://github.com/Chinano9/kizuna-editor/issues) del repositorio.

Supuestos considerados:

- Estimaciones en **días hábiles por persona** (1 desarrollador por issue).
- Equipo de **5 desarrolladores** sin mucha experiencia en **Go** y **Svelte** (Wails), pero con experiencia en otros lenguajes y tecnologías, por lo que se incluye tiempo de aprendizaje/curva de escritura.
- El backend de **Kizuna Cloud** (`/server`) está prácticamente sin implementar (solo `go.mod`), lo que impacta las issues de nube (#12–#16, #20–#23, #27).
- Ya existe la infraestructura de grabación/reproducción de audio en Go (issue cerrada #2) y bindings de pistas (AddTrack/DeleteTrack/UpdateTrack), lo que aligera #9, #10 y #26.
- La asignación **Developer (num)** reparte la carga por temas y dependencias para que ningún desarrollador acumule demasiadas tareas en paralelo. Objetivo: balancear ~12–15 días probables por persona.

## Tabla

| Título | Se resuelve después de | Óptimo | Probable | Pesimista | Developer (num) |
|---|---|---|---|---|---|
| #6 — Creación de un álbum vacío | | 0.5 d | 1.5 d | 3 d | Dev 3 |
| #7 — Asignar un proyecto a un álbum | 6 | 0.5 d | 1.5 d | 3 d | Dev 3 |
| #8 — Explorar álbumes | 6, 7 | 1 d | 2 d | 4 d | Dev 3 |
| #9 — Grabación rápida | | 1 d | 3 d | 6 d | Dev 4 |
| #10 — Bandeja de ideas | 9 | 1 d | 2.5 d | 5 d | Dev 4 |
| #11 — Transformar idea en proyecto | 9, 10, 26 | 1 d | 3 d | 6 d | Dev 4 |
| #12 — Inicio de sesión | | 2 d | 5 d | 8 d | Dev 1 |
| #13 — Subir a la nube | 12 | 1.5 d | 4 d | 7 d | Dev 1 |
| #14 — Consumir de la nube | 12, 13 | 1.5 d | 4 d | 7 d | Dev 1 |
| #15 — Compartir proyecto | 12, 13 | 1 d | 3 d | 6 d | Dev 2 |
| #16 — Check-out de proyecto | 12 | 1 d | 3 d | 6 d | Dev 2 |
| #17 — Quitar de álbum | 6, 7 | 0.5 d | 2 d | 4 d | Dev 3 |
| #18 — Editar álbum | 6, 8 | 1 d | 2.5 d | 5 d | Dev 3 |
| #19 — Eliminar idea rápida | 9, 10 | 0.5 d | 1.5 d | 3 d | Dev 4 |
| #20 — Ciclo de sesión | 12 | 0.5 d | 1.5 d | 3 d | Dev 1 |
| #21 — Visualización de cambios | 13, 14 | 1 d | 3 d | 6 d | Dev 2 |
| #22 — Liberación de candado | 13, 16 | 0.5 d | 2 d | 4 d | Dev 2 |
| #23 — Reintento y carga de contenido | 13, 14 | 1.5 d | 4 d | 8 d | Dev 2 |
| #24 — Internacionalización (i18n) | | 1 d | 3 d | 6 d | Dev 3 |
| #25 — Editor con renderizado AlphaTex | | 2 d | 5 d | 10 d | Dev 5 |
| #26 — Gestión central (CRUD) de Pistas | | 1 d | 3 d | 6 d | Dev 5 |
| #27 — Cifrado Zero-Knowledge | 13, 14 | 2 d | 5 d | 10 d | Dev 5 |
| #28 — Tolerancia a fallos en grabación | | 1.5 d | 4 d | 8 d | Dev 4 |
| #29 — Autoguardado silencioso del editor | 25 | 0.5 d | 1.5 d | 3 d | Dev 5 |

## Carga estimada por desarrollador (días probables)

| Developer | Issues | Carga probable |
|---|---|---|
| Dev 1 (nube base) | #12, #13, #14, #20 | 14.5 d |
| Dev 2 (colaboración/UX nube) | #15, #16, #21, #22, #23 | 15 d |
| Dev 3 (biblioteca e i18n) | #6, #7, #8, #17, #18, #24 | 12.5 d |
| Dev 4 (audio e ideas) | #9, #10, #11, #19, #28 | 14 d |
| Dev 5 (editor y seguridad) | #25, #26, #27, #29 | 14.5 d |

## Orden sugerido de ejecución

1. **Base local**: #6, #7, #8, #9, #10, #17, #25, #26.
2. **Base de nube**: #12 → #13 → #14.
3. **Colaboración**: #15, #16, #20, #22.
4. **Audio/editor avanzado**: #11, #19, #28, #29.
5. **Seguridad y pulido**: #21, #23, #24, #27.