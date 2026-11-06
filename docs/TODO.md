# Revisión y mejoras de `go-kickstart`

> **Seguimiento:** marca `[x]` en la columna **Completado** (o en la casilla) cuando el punto esté resuelto. `[ ]` = pendiente.

Lista de seguimiento de la revisión del código, migraciones, configuración, plantillas y documentación. Las propuestas son orientativas; cada ajuste puede resolverse con una implementación más sencilla. Las descripciones de los hallazgos conservan el contexto original.

**Verificación del plan completo**

- Tests existentes con `go test -race ./...`: correctos en SQLite, PostgreSQL y MySQL (contenedores locales dedicados).
- `go vet ./...`, `golangci-lint run ./...` y compilación: correctos, lint sin avisos.
- sqlc regenerado al cambiar queries y nombres; código generado ignorado, sin ediciones manuales.
- Escaneo `govulncheck@v1.9.0`: cero vulnerabilidades alcanzables; advertencia de módulo `GO-2026-5932` en `golang.org/x/crypto/openpgp`, paquete que la aplicación no importa. Los paquetes importados no tienen hallazgos en este escaneo.
- Comprobación manual de flags > entorno > YAML para ambos TTL, constructor y contextos inválidos, permisos/URIs/symlinks de SQLite y copias de locales.
- HTTP: límite de cuerpo, campos desconocidos, JSON extra, localización, prefijo, 401/403/404/405/500, `no-store`, `Allow`, estáticos, redirecciones y `ResponseController`.
- Autorización: un JWT sin roles obtiene permiso tras asignarlo en la base de datos; revocación invalida access y refresh anteriores.
- Upgrade desde `20261105002`: conserva sesiones y solicitudes válidas en los tres motores; SQLite descarta solicitudes huérfanas y el borrado aplica cascade.
- Fallos de migración con fixtures temporales ya retirados: PostgreSQL/SQLite revierten DDL y versión; MySQL conserva DDL parcial y marcador, y bloquea el reintento.

---

## 1. Hallazgos de mayor impacto

### 1.1 Las variables de entorno documentadas con guion no funcionan — Urgencia: **Alta**

**Completado:** [x]

**Resolución:** Se normalizan puntos y guiones a `_`. Verificados ambos TTL y la precedencia flags > entorno > archivo mediante el ejecutable.

**Dónde:** `cmd/root.go:77`, `contrib/app.example.yaml:28-31`.

`viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))` solo sustituye puntos. Las claves:

- `auth.access-token-ttl`
- `auth.refresh-token-ttl`

quedan como `APP_AUTH_ACCESS-TOKEN-TTL` y `APP_AUTH_REFRESH-TOKEN-TTL`, pero el ejemplo documenta `APP_AUTH_ACCESS_TOKEN_TTL` y `APP_AUTH_REFRESH_TOKEN_TTL`. Esas últimas se ignoran.

Lo verifiqué: con la variable con guion el valor cambia; con guion bajo se queda en el valor por defecto.

**Cómo atacarlo:** añadir el guion al reemplazo:

```go
strings.NewReplacer(".", "_", "-", "_")
```

o renombrar las claves a `auth.access_token_ttl`/`auth.refresh_token_ttl`. Añadir una prueba pequeña de precedencia de configuración.

---

### 1.2 `NewHandler` no valida la base de datos y usa `panic` de forma inconsistente — Urgencia: **Alta**

**Completado:** [x]

**Resolución:** `NewHandler` devuelve `(*Handler, error)` y valida DB y dependencias obligatorias. Remitente y DSN se validan una sola vez en su constructor; llamadas y tests existentes adaptados.

**Dónde:** `handlers/handler.go:55-81`.

Valida logger, localizer, authenticator, mailer, remitente y OTP, pero no valida `opts.DB`. Además:

- `NewHandler` entra en `panic`.
- `NewAuthenticator`, `i18n.NewLocalizer`, `otp.NewSigner` devuelven error.
- `cmd/server.go:74` valida `mail.from` y luego `NewHandler` vuelve a parsearlo y puede entrar en `panic`.
- `cmd/server.go` comprueba DSN vacío y `provider.Open` también lo comprueba.

Para una plantilla, esto es una trampa: un fallo de configuración debería producir un error claro al arrancar, no un panic genérico.

**Cómo atacarlo:** hacer que `NewHandler` devuelva `(*Handler, error)`, validar todas las dependencias obligatorias (incluida `DB`) y centralizar la validación del remitente en un solo sitio. Actualizar llamadas y tests. Mantener `panic` solo para invariantes de programación, no para configuración.

---

### 1.3 `expiresIn` puede devolver valores negativos — Urgencia: **Media**

**Completado:** [x]

**Resolución:** Se calcula `now` una sola vez para ambos tokens y las duraciones vencidas se limitan a cero.

**Dónde:** `handlers/auth.go:260-267`; contrato en `docs/api.yaml:277-282`.

```go
return expiresAt - now
```

Si el token caduca entre la emisión y la respuesta, el resultado puede ser negativo, pero el OpenAPI declara `minimum: 0`.

**Cómo atacarlo:** usar `max(0, expiresAt-now)` y calcular `now` una sola vez para ambas duraciones. Añadir un caso límite si se aceptan tests.

---

### 1.4 Los errores JSON de middleware no envían `Cache-Control: no-store` — Urgencia: **Media**

**Completado:** [x]

**Resolución:** La primera pasada añadió `no-store` al middleware y reutilizó el escritor en la raíz. El plan posterior centralizó lectura y escritura JSON en `internal/httpx`, compartido por handlers y middleware.

**Dónde:** `middleware/internal.go:10-18` frente a `handlers/internal.go:29-34`; `docs/api.yaml:7` promete `no-store` en respuestas JSON.

Las respuestas 401/403/500 generadas por autenticación/permisos pueden ser cacheadas por proxies, mientras que las de handlers no.

**Cómo atacarlo:** añadir `Cache-Control: no-store` en `writeJSONError` o, mejor, crear un helper interno compartido para escribir JSON de error y usarlo desde handlers y middleware. Evitar duplicar dos escritores de JSON con cabeceras distintas.

---

### 1.5 `auth_id` tiene formatos y longitudes distintas entre backends — Urgencia: **Media**

**Completado:** [x]

**Resolución:** `database.NewAuthID` genera 256 bits en hexadecimal para creación y revocación en los tres motores. Migración nueva amplía MySQL (cuentas y refresh tokens); los valores existentes se conservan para mantener sesiones.

**Dónde:**

- PostgreSQL: `gen_random_uuid()::text` → 36 caracteres.
- MySQL: `UUID()` y columna `VARCHAR(36)` → 36 caracteres.
- SQLite: `randomblob(32)` → 64 caracteres hexadecimales.
- `auth/authenticator.go:172`: `secrets.RandStr(128)` → 32 caracteres hexadecimales.

Hoy todo cabe, pero es inconsistente y frágil. Si se cambia la generación a 256 bits en SQLite/PostgreSQL, MySQL no podrá almacenar el valor con `VARCHAR(36)`.

**Cómo atacarlo:** estandarizar el formato. Recomendación: 32 bytes aleatorios en hexadecimal (64 caracteres) en todos los backends, ampliar MySQL a `VARCHAR(64)` y añadir una migración. Alternativa: usar UUID en todos. Probar emisión, refresh y revocación en los tres motores.

---

## 2. Dónde reducir líneas de código

| # | Zona | Repetición / oportunidad | Cómo atacarlo | Ahorro aprox. | Urgencia | Completado |
|---|---|---|---|---|---|---|
| 1 | `cmd/serve.go:23-61` | Cada ajuste aparece 3 veces: `viper.SetDefault`, definición de flag y `viper.BindPFlag`. Son 12 ajustes y 36 declaraciones. Además los nombres de flag y clave no siguen un patrón uniforme. | Definir una tabla `setting{key, flag, usage, def}` y recorrerla para defaults, flags y binding. Un `switch` sobre el tipo del default permite usar `String/Int/Bool/Duration`. | 20–30 líneas | **Media** | [x] |
| 2 | `handlers/auth.go` y `handlers/account.go` | Cinco bloques casi idénticos de `type req`, `var req`, `decodeJSON` y `writeError`: login, verify, refresh, change email y confirm. | Función genérica libre `decodeBody[T any](h *Handler, w, r, key, msg) (T, bool)` o `decodeInto`. Mover los tipos de request a nivel de paquete. | ~35 líneas | **Media** | [x] |
| 3 | `handlers/account.go:41-67` y `164-190` | El bloque “obtener identidad + `SelectAccountBySub` + 404/500” está duplicado. | Helper `h.loadAccount(w, r, sub) (*database.Account, bool)` que escriba los errores. Calcular `subject := strconv.FormatInt(sub, 10)` una sola vez por handler. | ~20 líneas | **Media** | [x] |
| 4 | `database/postgres/queries`, `database/sqlite/queries`, `database/mysql/queries` | Muchos wrappers hacen variable temporal + `if err != nil` + `return`, y las conversiones a modelos de aplicación se repiten en los tres backends. | Retornos directos donde no haya lógica extra. Añadir conversores pequeños `toAccount`, `toRefreshToken`, `toLoginRequest`, `toEmailChangeRequest`. Mantener separado el mapeo de errores específico de cada driver. | 40–60 líneas | **Media/Baja** | [x] — conversiones explícitas entre structs compatibles verificadas por el compilador, sin helpers genéricos |
| 5 | `database/postgres/migrations.go`, `database/sqlite/migrations.go`, `database/mysql/migrations.go` | El bucle de migraciones pendientes y `ensureMigrationTable` están replicados. | Extraer un runner común en `database` con callbacks para leer/escribir versión y ejecutar SQL; conservar el flujo especial de MySQL con `schema_migration_pending`. | ~25 líneas | **Media** | [x] — recorrido común transaccional para PostgreSQL/SQLite; MySQL conserva su marcador y DDL explícitos |
| 6 | `middleware/authentication.go` | Los argumentos de log `status`, `method`, `path` se repiten en casi todas las ramas; varias líneas superan 100 caracteres. | Helper `requestLogArgs(r, status, extra...)` y usarlo en autenticación y permisos. | ~10 líneas + legibilidad | **Baja** | [x] |
| 7 | `i18n/en.go`, `i18n/es.go` | Cuatro claves muertas: `err.missing_email_change`, `err.missing_login`, `err.select_email_change`, `err.select_login`. Además cada archivo registra la locale en un `init()`. | Eliminar las claves no usadas y declarar `var Locales = map[string]i18n.Locale{"en": en, "es": es}` en `i18n.go`, quitando los `init()`. | ~14 líneas | **Baja/Media** | [x] |
| 8 | `templates/base/body.templ`, `templates/base/head.templ` | `h(tr, active)` no usa ninguno de sus dos parámetros; `BodyParams.Active` se pasa desde `handlers/root.go` pero no se usa; hay HTML comentado que referencia un paquete `routes` inexistente. | Eliminar parámetros/campos no usados, borrar los comentarios muertos y renombrar `h/m/f` a `header/main/footer`. Si se quiere navegación activa, implementarla de verdad. | 10–15 líneas | **Baja** | [x] |
| 9 | `handlers/handler.go:104-106` | `Add` es solo un alias de `AddHandler`; unas rutas usan uno y otras el otro. | Elegir una API canónica (`AddHandler`) y usarla en todas las rutas, o mantener `Add` como azúcar pero documentarlo y usarlo consistentemente para `http.HandlerFunc`. | 3–8 líneas | **Baja** | [x] |
| 10 | `cache/memory/memory.go:78-145` | `Set`, `Get` y `Delete` repiten validación de contexto, clave vacía y estado cerrado. | Extraer `validateKey`/`validateContext` (sin intentar abstraer el lock) para reducir duplicación sin perder claridad. | ~10 líneas | **Baja** | [x] |

**Criterio aplicado:** las estimaciones originales eran orientativas. Se priorizó eliminar duplicación sin esconder diferencias de drivers, transacciones o seguridad; las interfaces pequeñas, validaciones y migraciones nuevas también requieren líneas.

---

## 3. Modularidad y legibilidad

| # | Oportunidad | Cómo atacarlo | Urgencia | Completado |
|---|---|---|---|---|
| 1 | Unificar escritura/lectura JSON HTTP | Crear un paquete interno pequeño, por ejemplo `internal/httpx`, con `DecodeJSON`, `WriteJSON` y `WriteJSONError`. Debe incluir `Cache-Control: no-store` de forma uniforme. No convertirlo en un framework propio. | **Media** | [x] |
| 2 | Unificar el logger | `handlers.Logger` tiene cuatro métodos, `middleware.Logger` dos y `cmd` usa `*slog.Logger`. Definir una interfaz compartida en `internal/logging` o usar `*slog.Logger` en toda la aplicación. | **Baja/Media** | [x] |
| 3 | Reducir el tamaño de `database.Querier` | Tiene ~25 métodos. Se puede componer por dominios: `AccountQuerier`, `SessionQuerier`, `OTPQuerier`, `RateLimitQuerier`, `MaintenanceQuerier`; `Querier` los embebe. Mejora testabilidad y permite que handlers/middleware dependan de interfaces pequeñas. | **Media** | [x] |
| 4 | Descriptor de propósito OTP | Hoy `login`/`email_change` se repiten como literales en scopes (`login.verify`, `email_change.verify`), claves de correo (`mail.<purpose>.subject`) y firmas. Un struct `otpPurpose{purpose, mailSubjectKey, mailBodyKey, issueScope, verifyScope}` evita erratas y permite validar claves. | **Media** | [x] — propósitos tipados; scopes y claves de correo derivados; sin descriptor mutable |
| 5 | Evitar N+1 al comprobar permisos | `RequirePermission` consulta roles y luego hace una consulta por rol. Añadir una consulta `AccountHasPermission(sub, permission)` con `EXISTS`/join, o `SelectAccountPermissionsBySub`, y reducir la interfaz `PermissionChecker` a un método. | **Media** | [x] |
| 6 | Separar la preparación de almacenamiento SQLite | `storage.go` mezcla parseo de URI, decisiones de creación, comprobación de permisos y manejo de Windows. Dividir en `parseStoragePath`, `ensurePrivateFile` y `validateStorageFiles`, y envolver errores con la ruta (`fmt.Errorf("... %q: %w", path, err)`). | **Baja/Media** | [x] |
| 7 | `auth.Verify` | Usa `switch err` contra errores de JWT, mientras `Refresh` y los handlers usan `errors.Is`. Unificar con `errors.Is` para tolerar errores envueltos. | **Baja** | [x] |
| 8 | Tipos de request/response | `sessionResponse` está a nivel de paquete, pero `loginRequest`, `verifyRequest`, `refreshRequest`, `changeEmailRequest` y `accountResponse` están dentro de funciones. Para una plantilla conviene un criterio único: todos a nivel de paquete junto al handler, o todos locales. | **Baja** | [x] |
| 9 | Opciones de `cache/memory` | `OptFunc` es exportado pero recibe `*memoryCache`, un tipo no exportado; un consumidor externo no puede crear opciones propias. Valorar `type Option interface { apply(*config) }` o documentar que solo se usan las opciones proporcionadas. | **Baja** | [x] |
| 10 | `serve` duplicado en Cobra | `rootCmd.RunE` y `serveCmd.RunE` llaman a `runServerFromConfig`. Decidir si `app` sin subcomando arranca el servidor por comodidad o si el comando canónico es `app serve`; documentarlo. | **Baja** | [x] |

---

## 4. Consistencia

| Tema | Inconsistencia | Cómo unificarlo | Urgencia | Completado |
|---|---|---|---|---|
| Variables de entorno | `APP_AUTH_ACCESS_TOKEN_TTL` documentada, pero Viper espera `APP_AUTH_ACCESS-TOKEN-TTL`. | Reemplazar también `-` por `_` o renombrar claves. | **Alta** | [x] |
| Archivo de configuración de ejemplo | Viper busca `app.yaml` (`cmd/root.go:73`), pero el ejemplo se llamaba `contrib/myserver.example.yaml` y su cabecera dice `# app.yaml`. | Renombrar a `contrib/app.example.yaml` o cambiar `SetConfigName`; actualizar README. | **Media** | [x] |
| Nombres de operaciones | `OncesertAccountByEmail` parece una errata de `Upsert`/`OnceInsert`. Se propaga a interfaz, adaptadores, queries SQL y tests. | Renombrar a `UpsertAccountByEmail` o `EnsureAccountByEmail` antes de publicar la plantilla; después es un cambio disruptivo. | **Media** | [x] |
| Iniciales Go | `Sqlite`, `NewSqlite`, `SqliteQuerier` frente a `Postgres`, `MySQL`, `MysqlQuerier`. | Usar `SQLite`, `NewSQLite`, `SQLiteQuerier` y `MySQLQuerier`; actualizar AGENTS.md. | **Media** | [x] |
| Iniciales en modelos | `OtpHash`, `Otp` deberían ser `OTPHash`, `OTP` en Go; los tags JSON pueden seguir siendo `otp`. | Renombrar campos de modelos y adaptadores; valorar `rename` en sqlc para código generado. | **Media/Baja** | [x] |
| Registro de rutas | `auth`/`root` usan `h.Add`; `account`/`static` usan `h.AddHandler`. | Elegir una API canónica. | **Baja** | [x] |
| Constructores | `NewHandler` hace panic; `NewAuthenticator`, `NewLocalizer`, `NewSigner` devuelven error; `initConfig` usa `cobra.CheckErr`. | Constructores de aplicación devuelven error; panic solo para invariantes internas. | **Alta** | [x] |
| Validación duplicada | `mail.from` se valida en `cmd/server.go` y en `NewHandler`; DSN vacío en `cmd/server.go` y `provider.Open`. | Validar una sola vez, preferiblemente en el constructor que posee el dato, y propagar error. | **Media** | [x] |
| Cabeceras JSON | Handlers ponen `Cache-Control: no-store`; middleware no; la raíz JSON escribe manualmente y tampoco. | Helper JSON único con cabeceras consistentes. | **Media** | [x] |
| Respuesta JSON de la raíz | `handlers/root.go:53-56` usa `json.NewEncoder` directamente en lugar de `writeJSON`. | Usar `writeJSON` para heredar cabeceras y formato. | **Baja** | [x] |
| Esquema entre backends | `account_email_change_requests` tiene FK con cascade en MySQL, FK sin cascade en PostgreSQL y sin FK explícita en SQLite. | Añadir `ON DELETE CASCADE` donde aplique o documentar que el handler borra manualmente; alinear migraciones. | **Media** | [x] |
| `auth_id` | UUID de 36 caracteres en PostgreSQL/MySQL, 64 hex en SQLite, 32 hex tras `RevokeAll`. | Estandarizar formato y longitud; ampliar MySQL si hace falta. | **Media** | [x] |
| Vista `accounts_meta` | PostgreSQL/SQLite usan `SELECT a.*` (incluye `auth_id`); MySQL enumera columnas (no incluye `auth_id`). Las queries seleccionan columnas explícitas, así que la vista no aporta. | Eliminar la vista y consultar `accounts`, o mantener columnas idénticas en los tres motores. | **Baja/Media** | [x] |
| Idempotencia de migraciones | `20261104000_rate_limits` y `20261105000_expiration_indexes` usan `CREATE INDEX` sin `IF NOT EXISTS`; otras migraciones sí lo usan. MySQL no soporta esa cláusula. | Usar `IF NOT EXISTS` donde el motor lo permita; documentar la limitación de MySQL en la recuperación. | **Baja** | [x] — decisión: conservar archivos publicados y fallar ante divergencia de esquema |
| i18n | Claves muertas, `err.invalid_email_change_body` frente a `err.invalid_email_confirm_body`, `init()` por locale y `Locales` global mutable. | Limpiar claves, unificar nombres, registrar locales en un solo sitio. | **Media/Baja** | [x] — locales copiados; se conservan claves estables de solicitud y confirmación |
| Español | `codigo`, `sesion`, `verificacion`, `invalido` sin tildes; `misspell` las marca como errores en inglés. | Si el estilo ASCII es intencional, documentarlo y configurar `misspell`; si no, corregir tildes. | **Baja** | [x] |
| Ejemplos OpenAPI | `refresh_token_expires_in: 5184000` (60 días) no coincide con el default de 30 días (`2592000`). El `refresh_token` de ejemplo parece un JWT, pero se genera como cadena aleatoria. | Corregir ejemplos en `docs/api.yaml:77,106`. | **Baja** | [x] |
| Documentación de caché | AGENTS.md propone cachear listas de roles y comprobaciones de permisos, pero luego indica que roles/permisos se saltan la caché. | Decidir si se implementará o eliminar esas recomendaciones para que la guía no se contradiga. | **Baja** | [x] |
| Placeholders de plantilla | Módulo y comando `app`, `package.json` `app`, título `App`, pero ejemplo `myserver`, README `go-kickstart` y textos Cobra de ejemplo. | Elegir un nombre/placeholder único y actualizar README, `package.json`, Cobra y configuración de ejemplo. | **Baja** | [x] |
| Cabeceras de licencia | `cmd/root.go` tiene la GPL completa, `cmd/serve.go`/`logging.go` solo copyright y la mayoría de archivos no tienen cabecera. | Aplicar la misma política a todos los archivos o eliminar las cabeceras parciales. | **Baja** | [x] |
| Contexto nulo | `cache/memory` acepta `nil`; `mailer/noop` devuelve `ErrNilContext`. | Documentar o normalizar la política. | **Baja** | [x] |
| Migración bootstrap | `20260503000_schema_migrations.sql` no usa `.up.sql`, a diferencia del resto. | Es intencional porque se aplica aparte, pero conviene documentarlo junto al formato de nombres. | **Baja** | [x] |

---

## 5. Quick wins

- [x] **Comprobar errores al cerrar la caché** — Urgencia **Media**:
  - `cmd/server.go:84`: cerrar el cache con log de error, igual que se hace con la base de datos.
  - `cache/memory/memory_test.go`: usar un helper con comprobación de error en los cierres, también en los concurrentes. Se conserva `defer` para cerrar dentro de la burbuja de `synctest`.
- [x] **`prealloc` en `handlers/internal.go:37`** — Urgencia **Baja**: `logArgs := make([]any, 0, 6+len(args))`.
- [x] **`package.json`** — Urgencia **Baja/Media**:
  - `npm run redocly` usa `mkdir -p` (no portátil en Windows) y oculta todos los errores con `>/dev/null 2>&1`; se eliminaron `mkdir -p` y las redirecciones. Redocly crea el directorio de salida y sus errores quedan visibles.
  - El script `test` es un placeholder que siempre falla; cámbialo por `go test ./...` o elimínalo.
  - `main: "index.js"` apunta a un archivo inexistente; para un proyecto solo de documentación, considera `"private": true` y quitar `main`.
- [x] **README/configuración** — Urgencia **Baja**: corregir el nombre del archivo de ejemplo y explicar que `app serve` es opcional porque `app` ya arranca el servidor.
- [x] **`docs/api.yaml`** — Urgencia **Baja**: corregir TTL y ejemplo de refresh token.
- [ ] **Cobertura automatizada adicional** — Pendiente de petición explícita conforme a AGENTS.md: añadir tests permanentes para configuración, middleware, routing y almacenamiento SQLite. Los escenarios indicados arriba ya se comprobaron manualmente; se adaptaron y ejecutaron los tests existentes. No se marca completado porque las comprobaciones temporales no sustituyen esa cobertura.

---

## 6. Plan de implementación por commits

El orden estabiliza primero los contratos, después cambia comportamiento y esquema, y finalmente simplifica consumidores y documentación. Cada commit actualiza este seguimiento y ejecuta generación pertinente y tests existentes.

1. [x] `151554f` — **Contratos de base de datos:** `EnsureAccountByEmail`, iniciales `SQLite`/`MySQL`/`OTP`, interfaces por dominio y adaptadores más directos.
2. [x] `fa55558` — **Permisos:** una consulta `EXISTS` por cuenta y permiso; mantener autorización fuera de caché.
3. [x] `44f3f82` — **Generaciones de sesión:** generar identidades nuevas de 256 bits en la aplicación, ampliar columnas MySQL y conservar identidades y sesiones existentes.
4. [x] `55e7209` — **Ciclo de vida de cuentas:** FK con cascade para cambios de correo, preservar solicitudes válidas y eliminar la vista innecesaria `accounts_meta` mediante migraciones nuevas.
5. [x] `924df3d` — **HTTP y handlers:** JSON compartido, logger mínimo compartido, validación de cuerpos y carga de cuentas reutilizables; documentar la distinción entre `Add` y `AddHandler`.
6. [x] `724a7f1` — **OTP e idiomas:** propósitos tipados y scopes derivados, presupuestos de verificación centralizados, locales sin mapa público mutable y español con tildes; mantener claves estables cuando ya distinguen bien su propósito.
7. [x] `5110b81` — **Configuración y almacenamiento:** tabla de ajustes, validación de caché compartida con contrato de contexto explícito, y preparación SQLite dividida en funciones pequeñas.
8. [x] `99fa49d` — **Migraciones:** compartir solo el recorrido de migraciones donde reduzca duplicación y mantenga visibles locks, transacciones y recuperación MySQL. Conservar migraciones publicadas sin alteraciones.
9. [x] **Documentación y cierre:** nombres y licencias coherentes, guía de extensión actualizada, revisión de todas las filas del TODO y verificación con SQLite, PostgreSQL y MySQL.

Las propuestas se evalúan por su beneficio: no se introducirán abstracciones solo para reducir líneas. Las decisiones de conservar código se explicarán en el punto correspondiente. No se añadirán tests nuevos sin petición explícita, según AGENTS.md; se adaptarán los existentes y se comprobarán manualmente los escenarios sin cobertura.

## 7. Fixes posteriores al plan

- [x] **Secretos según el modo de ejecución** (`c6fa6f5`): ambos requieren al menos 32 bytes fuera de desarrollo. `--dev` acepta valores vacíos o cortos con warnings; un OTP secret vacío usa una clave temporal. Los valores no vacíos deben ser distintos. README y ejemplo actualizados.
- [x] **Validación del prefijo de rutas**: `NewHandler` rechaza prefijos que provocarían un panic de ServeMux (`/api//v1`) o romperían los estáticos (`/v%31`). Solo se admiten rutas literales limpias; se conservan la normalización de barra inicial/final y los parámetros en rutas de endpoints. Documentación actualizada.

Verificación del prefijo: tests existentes con `-race`, vet, lint y compilación correctos; 20 formas inválidas y 9 válidas comprobadas manualmente, además del error de configuración en el ejecutable CLI. No se añadieron tests permanentes.
