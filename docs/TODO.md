# Plan de seguridad, rendimiento y consistencia

Revisión del 14 de noviembre de 2026 sobre `f3976c9`. Este documento propone cambios; no implica que estén implementados. El orden prioriza seguridad, después rendimiento y finalmente simplificación. Las dependencias indicadas evitan modificar dos veces una misma solución.

## Alcance y resultados

Se revisaron autenticación y OTP, autorización, cookies/CSRF/CORS, adaptación web de los handlers, TypeScript/templ, consultas y mantenimiento SQL, caché, límites HTTP, nginx, Docker y CI. Se inspeccionaron también las implementaciones instaladas de JWT, validación de correo y transporte SMTP.

Validaciones realizadas:

- `go test -race ./...` con Go 1.26.11: pasó. SQLite incluido; PostgreSQL/MySQL no se ejecutaron porque sus variables `TEST_*_DSN` no estaban configuradas.
- `govulncheck@v1.9.0 ./...`: sin vulnerabilidades conocidas detectadas.
- `npm audit --json --ignore-scripts`: cero avisos.
- `npm run lint` y `npx tsc --noEmit`: pasaron.
- Contenedor nginx temporal usando la configuración de despliegue y un upstream de diagnóstico: ocho POST consecutivos a `/auth/login` produjeron `204,204,204,429,429,429,429,429`. Ocho PATCH a cada ruta de cambio de correo produjeron ocho 204. Esto prueba la omisión del límite del proxy, no una omisión de las cuotas del backend.
- Diagnóstico temporal con el autenticador real y SQLite: emitir R0, renovarlo a R1, revocar R0 devuelve `nil`; el access token de R1 sigue válido y R1 todavía puede renovarse. No se imprimieron credenciales ni se usaron cuentas reales.

No se efectuó una prueba de penetración del despliegue real ni una carga representativa de producción. Los costes señalados por lectura de código no son cuellos de botella medidos. Un resultado limpio de dependencias no certifica la ausencia de vulnerabilidades lógicas. No se añadieron pruebas permanentes ni se modificó código funcional en esta revisión.

Controles existentes que deben conservarse:

- JWT con HS256 verificado, comprobación de caducidad y consulta de sesión/generación sin caché en cada autenticación. Permisos consultados en SQL, no autorizados por los roles potencialmente antiguos del JWT.
- Refresh aleatorios almacenados como SHA-256; rotación transaccional; OTP como HMAC con secreto independiente y consumo atómico. Cuotas persistentes de emisión/verificación que no se reinician al reemplazar un OTP.
- Consultas parametrizadas, paginación de sesiones acotada e índices de sesión/expiración ya existentes. No se identificó una inyección SQL o XSS explotable en los recorridos revisados.
- Cookies HttpOnly/Secure/SameSite=Strict en producción; API exclusivamente Bearer; CSRF en POST web; GET no rota tokens. El cliente no necesita leer tokens ni trasladarlos a localStorage.
- Escape de templ, ausencia de `innerHTML`/`eval` en el cliente propio, límites de cuerpos/cabeceras y tiempos HTTP, SMTP con TLS y validación de certificado.
- Caché limitada y desactivada por defecto, almacenamiento SQLite privado, migraciones con bloqueo y detección de migraciones MySQL incompletas.

## Orden de aplicación

### 1. Corregir y verificar los límites del proxy

- [x] **Prioridad alta; omisión corregida.** Se incluyó PATCH en `$limit_ip` en `deploy/nginx.conf` y `contrib/nginx.example.conf`, compartiendo presupuesto con POST. Los endpoints de cambio y confirmación de correo ya consumen el límite por IP. GET/HEAD/OPTIONS siguen sin consumir presupuesto. La guía identifica los métodos reales de la API.
- [x] Comprobar la matriz método/ruta contra el catálogo Go, incluyendo `APP_ROOT`, escapes de URL y barras normalizadas. Validado en ambos ejemplos mediante contenedores temporales: límites compartidos POST/PATCH, rutas prefijadas, `%63`, barras repetidas y segmentos `..`; GET/HEAD/OPTIONS funcionan incluso con el presupuesto agotado. Cabeceras CF/XFF falsas no evaden el límite en el escenario de ingreso simulado. `go test ./...` pasó. No se cambiaron los métodos de la API.
- [ ] Documentar que todo POST `/login` comparte el presupuesto de emisión: verificar, renovar y salir pueden recibir 429 bajo la misma IP/NAT. Medir esa interferencia antes de ajustar tasas. No clasificar seguridad mediante un campo de formulario o cabecera que el cliente pueda falsificar para eludir el límite; conservar las cuotas persistentes por identidad.
- [ ] Validar en Railway real que el edge sustituye `X-Real-IP`, que solo ese ingreso alcanza nginx y que `CF-Connecting-IP` solo se utiliza con un peer Cloudflare verificado. Ensayar cabeceras falsas por dominio Railway directo, dominio Cloudflare e IPv6. Mantener actualizada la lista de redes CF. No describir este control como «por dispositivo»: es por IP pública y por instancia nginx.

**Archivos:** ambas configuraciones nginx y `docs/reverse-proxy.md`. **Aceptación:** PATCH alcanza 429; las cuotas SQL siguen funcionando entre réplicas; peticiones legítimas GET y preflight no se limitan; falsificar cabeceras desde el ingreso público no cambia de presupuesto. La documentación de Railway identifica `X-Real-IP`, pero eso no sustituye verificar la frontera de confianza del despliegue. [Railway](https://docs.railway.com/networking/public-networking/specs-and-limits), [nginx limit_req](https://nginx.org/en/docs/http/ngx_http_limit_req_module.html).

### 2. Cerrar las ambigüedades de revocación y renovación

- [x] **Revocación tras rotación implementada.** `auth.Authenticator.Revoke` resuelve hashes actuales o consumidos, coordina por cuenta y revoca por sesión estable. La migración `20261114000` conserva únicamente hash, identidad/generación y caducidad; no modifica sesiones existentes. No se pueden reconstruir hashes consumidos antes del despliegue.
- [x] La revocación de un refresh identificable usa el mismo orden de bloqueo cuenta→sesión que la renovación. Los hashes desconocidos/no retenidos siguen siendo un no-op idempotente para conservar el contrato; no se afirma que eso pruebe una revocación. La documentación distingue ambos casos.
- [x] La referencia de consumo se guarda en la misma transacción que consumo y reemplazo, hasta la caducidad original de refresh/access; para refresh legacy sin caducidad se usa un TTL configurado desde el consumo o la caducidad de access, si es posterior. Se limpia por lotes de 100 bajo el plazo del mantenimiento y se borra al eliminar la cuenta. El coste es una fila por rotación hasta su caducidad, a medir en el paso 9.
- [x] Política de replay definida: los hashes consumidos permiten revocar pero nunca renovar; un duplicado devuelve 401 y no revoca automáticamente al sucesor. Se evita cerrar sesiones por reintentos legítimos, conservando la limitación de no contener automáticamente a quien ya tenga el sucesor. Una respuesta de renovación perdida todavía exige login. No se añadió caducidad absoluta ni recuperación de replay.
- [x] Logout distingue errores de `Verify`: los fallos de infraestructura devuelven 503 (504 si vence el contexto de la petición) y conservan cookies; los tokens inválidos/expirados siguen el camino de revocación por refresh. Diagnóstico temporal con y sin refresh comprobó errores de dependencia, credenciales inválidas/expiradas y cierre mediante access válido sin refresh.
- [ ] Considerar una caducidad absoluta opcional basada en `created_at`: actualmente cada rotación extiende el refresh y una sesión activa puede durar indefinidamente. Es una decisión de producto/seguridad, no un fallo de expiración del token actual. Mantener la política actual por defecto hasta acordar el cambio.

**Archivos:** `auth/authenticator.go`, `handlers/browser.go`, interfaces/adaptadores SQL, `docs/sessions.md`, `docs/browser.md`. **Dependencias:** cerrar este contrato antes de los pasos 8 y 10. Si requiere esquema, migraciones nuevas coordinadas para los tres motores; conservar sesiones existentes y prever despliegue sin escritores antiguos.

**Aceptación:** logout contra refresh en ambos órdenes, revocación individual y global, access expirado/ausente, error de BD, credenciales antiguas, respuesta perdida y dos réplicas. Una sesión ajena jamás puede revocarse; no se borra una cookie nueva por un fallo tardío. Ejecutar la suite existente y documentar las reproducciones adicionales necesarias antes de implementar.

**Validación de la retención:** `go generate ./database` y `go test -race ./...` con SQLite/PostgreSQL/MySQL pasaron. Diagnósticos temporales en los tres motores comprobaron revocación tras múltiples rotaciones, rechazo de renovación con el hash antiguo sin revocación automática, rollback de emisión, caducidad, limpieza de más de un lote y cascada al borrar la cuenta. También se comprobó POST `/app/login` en producción con refresh retenido y sin access: redirige, elimina cookies e invalida ambos tokens sucesores. No se añadieron pruebas permanentes; se actualizó la expectativa de versión en la prueba de migración existente.

### 3. Añadir protección del documento sin romper htmz

- [ ] **Prioridad media; defensa ausente, no XSS demostrado.** No hay CSP ni política anti-framing en las páginas propias. Añadir cabeceras coherentes en Go para que también apliquen sin nginx: `X-Content-Type-Options: nosniff`, `frame-ancestors 'self'` y, si se mantiene compatibilidad con clientes antiguos, `X-Frame-Options: SAMEORIGIN`. No usar `DENY` ni `frame-ancestors 'none'`: bloquearían el iframe de htmz.
- [ ] Diseñar CSP para la superficie propia con scripts de origen propio, `object-src 'none'`, `base-uri 'none'`, `form-action 'self'` y `frame-src 'self'`. Probar primero en Report-Only. Contemplar el favicon `data:` y el atributo `style` existente en `templates/home/home.templ`: no imponer una política que rompa su presentación ni añadir CSS nuevo sin autorización. Si es necesario permitir estilos inline temporalmente, restringir la excepción a estilos, nunca a scripts.
- [ ] Revisar Redocly por separado: el HTML generado contiene recursos/scripts distintos; una CSP global rígida puede romperlo. No relajar las páginas de login/home para acomodar la documentación.
- [ ] Mantener `Referrer-Policy: strict-origin` en el flujo de formularios, por la compatibilidad ya corregida con HTTP local. Configurar HSTS solo en el borde HTTPS de producción, después de verificar el dominio; no imponerlo en desarrollo ni activar `includeSubDomains` indiscriminadamente.
- [ ] Explicitar en el arranque que `--dev` deshabilita comprobaciones de origen y muestra OTP, incluso con secretos largos. Conservar la semántica solicitada de desarrollo y los overrides de logging; asegurar que los ejemplos de producción no lo habilitan.

**Aceptación:** un origen externo no puede enmarcar login/home; el iframe del mismo origen sigue funcionando; CSP no rompe modal, fechas, SVG, CSS, documentación, formularios ni raíz prefijada. SameSite reduce exposición a framing entre sitios, pero no reemplaza una política de ancestros. [CSP](https://developer.mozilla.org/en-US/docs/Web/HTTP/Guides/CSP), [frame-ancestors](https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/Content-Security-Policy/frame-ancestors).

### 4. Reducir exposición de datos y trabajo por entradas inválidas

- [ ] **Prioridad media.** El correo viaja en `/login?step=verify&email=...`; nginx hereda un access log que incluye la petición y su query. Registrar método, URI sin query, estado y duración, sin Cookie/Authorization/cuerpos. Revisar también retención y acceso a logs de Railway/Cloudflare. No registrar correos completos o entradas arbitrarias de hasta 1 MiB en errores; truncar/redactar los campos actuales de `handlers/auth.go` y `handlers/account.go`.
- [ ] Decidir si se requiere eliminar el correo de URL/historial. La corrección mínima es sanear logs. Quitar la query exige un mecanismo de estado seguro adicional o cambiar el flujo POST/redirect/GET; no introducirlo solo para ahorrar líneas ni afirmar que `no-store` borra el historial del navegador.
- [ ] Validar la forma del OTP en servidor antes de HMAC/SQL: seis dígitos ASCII, como ya pide el formulario. Mantener los límites actuales por identidad y definir explícitamente el tratamiento de entradas malformadas, sin reiniciar presupuestos. El correo ya tiene límites de longitud en la dependencia; no duplicar su parser.
- [ ] Aplicar límites de tamaño a credenciales solo con valores derivados de los tokens emitidos y de clientes existentes; evitar un límite arbitrario que rompa compatibilidad. Conservar el límite global de cuerpo y los errores localizados.

**Aceptación:** logs sin tokens/OTP en producción ni correo en queries; entradas grandes y malformadas se rechazan con coste acotado; API y formulario conservan respuestas documentadas. No añadir CAPTCHA o una cola de correo sin evidencia de necesidad.

### 5. Asegurar artefactos y privilegios del despliegue

- [ ] **Prioridad media; endurecimiento preventivo.** `templates/semantic.go` calcula un hash del contenido descargado para el nombre, pero no verifica un hash esperado. Añadir SHA-256 aprobado junto a la tag, límite de bytes por descarga y fallo antes de publicar archivos si no coincide. Mantener la descarga desde release minificado y las licencias. La huella del filename invalida caché; por sí sola no prueba integridad del proveedor.
- [ ] Fijar Actions e imágenes base por digest/SHA revisado, con un proceso regular de actualización; fijarlas y olvidarlas acumularía vulnerabilidades. Incorporar auditoría npm y escaneo de la imagen final en CI además del `govulncheck` existente. Distinguir dependencias de build de paquetes efectivamente desplegados, sin ignorar que los scripts de instalación ejecutan código durante el build.
- [ ] Para producción, documentar usuario SQL de ejecución con privilegios mínimos, migraciones mediante credenciales separadas y `auto-migrate=false` tras el job migrador. MySQL habilita `MultiStatements` para migraciones en el mismo conector usado por la app: evaluar limitarlo al camino migrador sin duplicar innecesariamente la conexión. No se encontró concatenación de SQL con datos de usuario; esto reduce privilegios, no corrige una inyección demostrada.
- [ ] Verificar TLS del DSN cuando la red de BD lo requiera, con validación del servidor, sin `skip-verify`. Documentar el supuesto de red privada si no usa TLS. Mantener Go sin privilegios y sin directorio de escritura; nginx/tini necesitan sus recursos de runtime. No quitar el entrypoint: actualmente supervisa dos procesos y preserva el orden de apagado.

**Aceptación:** checksum incorrecto no publica assets; instalación reproducible; imagen sin avisos explotables pendientes de revisar; usuario de ejecución no puede hacer DDL y `app migrate` sigue funcionando. No cambiar permisos reducidos de CI ni el flujo de publicación.

### 6. Establecer una línea base de rendimiento

- [ ] **Antes de optimizaciones estructurales.** Medir con `--dev=false`, secretos de prueba y BD aislada: `/home` con 1/100 sesiones, lectura de cuenta, login con mailer de prueba, renovación, errores y estáticos fríos/calientes. Comparar acceso directo y vía nginx, una y dos réplicas, y concurrencia dentro/fuera del tamaño del pool.
- [ ] Registrar p50/p95/p99, errores/429, consultas por operación, esperas del pool, CPU, asignaciones, bytes y antigüedad de filas vencidas. Separar latencia SMTP y SQL de render/JSON/compresión. Perfilar con herramientas locales; no publicar pprof sin autenticación.
- [ ] Fijar objetivo de latencia/capacidad según recursos reales. No inventar porcentajes de mejora ni aumentar pools a ciegas: `réplicas × máximo de conexiones` debe caber en la BD. SQLite tiene una conexión serializada deliberadamente.

**Resultado requerido:** tabla antes/después por cambio y carga reproducible. Las mejoras sin beneficio medible pueden descartarse si añaden complejidad. Mantener los presupuestos de abuso durante las mediciones, distinguiendo pruebas funcionales de pruebas de capacidad autorizadas.

### 7. Aprovechar los assets con hash y afinar compresión

- [ ] **Oportunidad confirmada por inspección.** `handlers/static.go` usa FileServerFS sin `Cache-Control` propio para CSS/JS fingerprinted. Añadir `public, max-age=31536000, immutable` únicamente a assets existentes e inmutables identificados por el manifiesto/hash, en respuestas válidas. No aplicar a `/login`, `/home`, API, HTML de documentación, directorios ni 404.
- [ ] Mantener HTML autenticado y tokens con `no-store`. No añadir caché compartida de páginas de sesión en nginx/Cloudflare. Comprobar que el despliegue gradual sirve los assets referenciados por HTML de ambas versiones: las imágenes actuales solo incluyen el bundle vigente. Caché immutable no resuelve el primer fetch enviado a una réplica de otra versión; conservar assets anteriores o publicarlos en almacenamiento común si se hacen despliegues mixtos.
- [ ] Medir gzip nivel 5 frente a niveles menores y el coste de recomprimir bundles. Solo si compensa, precomprimir assets durante build y servirlos con negociación correcta (`Vary`, HEAD, rangos, `gzip;q=0`); no duplicar ahora todo el árbol de estáticos entre Go y nginx.
- [ ] Verificar reutilización de conexiones al upstream con la versión efectiva de nginx antes de añadir `upstream keepalive`: los valores por defecto cambiaron en versiones recientes. No presentar su ausencia explícita como cuello de botella probado.

**Dependencias:** paso 5 para la integridad y 6 para medición. **Aceptación:** cambios de contenido cambian URL, cargas repetidas reutilizan caché, errores no quedan cacheados y una página autenticada nunca se sirve a otro usuario. [Caché HTTP](https://developer.mozilla.org/en-US/docs/Web/HTTP/Guides/Caching), [keepalive nginx](https://nginx.org/en/docs/http/ngx_http_upstream_module.html).

### 8. Compartir operaciones tipadas entre API y web si su coste lo justifica

- [ ] **Coste confirmado, impacto por medir.** `internal/browser.Call` serializa payloads, invoca un handler con un ResponseWriter en memoria y vuelve a decodificar JSON. `/home` construye JSON de sesiones y luego lo transforma en filas HTML; logout puede verificar el mismo access dos veces.
- [ ] Tras el paso 2, extraer únicamente operaciones compartidas a funciones/servicios tipados: verificar identidad, listar sesiones, emitir/renovar y revocar. Mantener adaptadores HTTP delgados que traduzcan resultados a JSON o HTML/cookies. No crear un framework de repositorios ni un servicio por cada función trivial.
- [ ] Mantener autorización, cuotas OTP, transacciones y clasificación de errores en un único camino. Reutilizar una identidad verificada dentro de la misma petición; nunca cachearla entre peticiones ni omitir la consulta de revocación.
- [ ] El consumo del OTP/creación de cuenta y la emisión de sesión hoy son dos transacciones: un fallo en la segunda consume el código sin entregar sesión. Evaluar unirlos con una operación transaccional pequeña al extraer el servicio, conservando cuotas fuera del rollback y sin mantener locks mientras se envía SMTP. Es una mejora de recuperación, no una reutilización actual del OTP. No reintentar emisión silenciosamente tras un resultado ambiguo.
- [ ] Eliminar `browser.Call`/`Reply` solo cuando todos sus consumidores hayan migrado. Si el ahorro no compensa el refactor, conservar el adaptador y tratar correctamente los errores de `json.Marshal` que hoy se ignoran.

**Aceptación:** mismos endpoints, métodos, estados, JSON, traducciones y metadatos de dispositivo; cookies solo en la superficie web; comparación de consultas/asignaciones con el paso 6. Este paso concentra la simplificación del adaptador y su optimización; no planificar después otro reemplazo paralelo.

### 9. Afinar SQL y mantenimiento sobre el diseño final de sesiones

- [ ] **Riesgo de capacidad, no saturación observada.** Login/email/refresh eliminan como máximo 1000 filas vencidas por tabla y minuto por instancia; rate limits se drenan en lotes de 100 dentro del plazo compartido de cinco segundos. Una llegada sostenida superior al drenaje puede acumular filas, especialmente repartida entre muchas identidades/IP. Medir backlog/antigüedad, y drenar también las otras tablas por lotes bajo el mismo plazo si hace falta.
- [ ] Usar EXPLAIN con cardinalidad representativa para limpieza y listado paginado. Ya existen índices por expiración y `(sub, created_at, session_id)`; no añadir duplicados. Revisar la selectividad de `access_expires_at` junto con `expires_at` después del diseño del paso 2.
- [ ] En varias réplicas, escalonar el mantenimiento si se observan contenciones; no introducir un líder distribuido sin necesidad. Cualquier retención de hashes consumidos del paso 2 debe entrar en este presupuesto.
- [ ] Medir round trips de `TakeRateLimit` en MySQL y locks por cuenta en refresh. Mantener atomicidad y el orden de bloqueo al optimizar. No eliminar cuotas, separar lectura/escritura sin transacción ni cachear autorización para ganar rendimiento.
- [ ] Mantener perfil caching desactivado salvo evidencia; su invalidación es local y no da consistencia inmediata entre réplicas. No añadir Redis, pooling SMTP ni colas globales sin una necesidad medida. Si SMTP domina, evaluar una cola acotada con semántica explícita de entrega y reintentos, sin resetear desafíos/cuotas.

**Dependencias:** 2, 6 y, si se ejecuta, 8. **Aceptación:** backlog converge, limpieza se cancela dentro del presupuesto, cuotas y revocación conservan consistencia, sin regresión p95 bajo carga. Consultas/migraciones coordinadas en los tres motores y generación SQL solo cuando corresponda.

### 10. Unificar htmz, formularios y estados del frontend

- [ ] **Consistencia solicitada; resolver en un único refactor.** Actualmente refresh usa botón `data-href` y formulario creado por JS; paginación usa enlace `data-htmz`; login/confirmación usan formulario; «restart» recarga la página. Unificar las acciones equivalentes dentro de `#browser-view`: enlace GET con `href` para navegación, formulario con `method/action` para envío y botón nativo asociado mediante `form` cuando la acción lo requiera. Preservar el aspecto pedido de enlaces y botones, sin CSS adicional.
- [ ] Usar `data-htmz` como opt-in común en enlaces/formularios compatibles, incluyendo «restart» del login. No añadirlo ciegamente al enlace de la página raíz: esa página actualmente no monta BrowserSurface ni carga el transporte; conservar navegación completa salvo que se adopte expresamente ese shell.
- [ ] Preferir un formulario GET declarado y un botón `form=...` para refresh a generar un formulario desde `data-href`. Revisar otros consumidores antes de eliminar la rama genérica de botones. Si debe conservarse por compatibilidad, hacer que delegue al mismo flujo de submit, sin otro mecanismo de envío ni otra política de busy/errores.
- [ ] Centralizar el ciclo de petición en un estado pequeño: elemento activador, iframe/solicitud actual, timer, liberación del Web Lock y limpieza. En el código actual el formulario temporal solo se asigna después de obtener el lock; si expira la espera, `finish()` no conoce ese formulario para retirarlo. Si se elimina la creación dinámica, eliminar también ese problema; si se conserva, registrar su limpieza desde el inicio. Evitar callbacks tardíos que actúen sobre una solicitud posterior.
- [ ] Normalizar y validar URL/target antes de modificar estado, incluyendo hash que no sea selector válido y `formaction/formmethod`; restaurar los atributos temporales de los enlaces. Mantener mismo origen, modificadores de clic, `_blank`, download, validación HTML y el helper que evita que `input name=action` oculte `form.action`.
- [ ] Mantener renovación/serialización de cookies en `session.ts` con un contrato pequeño hacia `htmz.ts`, y transporte/DOM en `htmz.ts`. Los locks son coordinación del cliente, no autorización. Conservar fallback sin Web Locks en HTTP local y no reintentar automáticamente un POST de efecto desconocido tras timeout.
- [ ] Mantener apertura/cierre de `<dialog>` como interacción local, no petición htmz. Conservar `commandfor` para botones y un helper pequeño para los enlaces de tabla que el usuario pidió mantener como `<a>`. Verificar soporte de los navegadores objetivo antes de quitar fallback. Unificar nombres de atributos sin intentar convertir todos los elementos en el mismo tipo HTML.
- [ ] Alinear URLs con el catálogo: `/home` y POST `/login` se registran como rutas raw y usan strings locales, mientras GET `/login` es nombrado. Añadir sus IDs y usar `h.AddRoute`/`links.URL` al actualizar templates. Mantener `--root`, 404 y la capacidad de quitar la superficie web; la validación de registro sigue siendo de inicio, no una garantía completa de compilación.

**Dependencias:** contratos de sesión del paso 2 y adaptadores finales del 8; respetar CSP del 3. **Aceptación:** misma UX y estados para login, verify, resume, refresh, paginación, restart y logout; busy en el control correcto y limpieza en éxito/error/cancelación/timeout; teclado, foco del modal, atrás/adelante, doble clic, dos pestañas, raíz prefijada y JS deshabilitado. Ninguna mutación pasa a GET ni se exponen tokens a JS. Ejecutar ESLint/TypeScript y regenerar el bundle con hash. No reescribir a fetch/HTMX/Datastar solo por reducir líneas.

### 11. Simplificar presentación únicamente donde conserve comportamiento

- [ ] `frontend/dates.ts` y `templates/home/dates.go` duplican tablas de meses/reloj. Evaluar `Intl.DateTimeFormat` con instancias reutilizadas por idioma/opciones para el cliente. Mantener `<time datetime>` y fallback UTC del servidor; sin conocer la zona del usuario, el servidor no puede reemplazar correctamente esta conversión local.
- [ ] Comparar salida en español/inglés, medianoche, cambio de año y horario de verano. Intl puede cambiar puntuación y abreviaturas: si conservar exactamente la presentación exige más código, mantener el formatter actual. No cambiar silenciosamente el formato por reducir líneas.
- [ ] La tabla genera hasta 100 modales y vuelve a renderizar la página completa al refrescar. Medir bytes/nodos/tiempo antes de sustituirlos por un modal compartido o respuestas parciales. Un modal compartido necesita asignar `session_id` y texto de forma segura con `textContent`, limpiar estado al cerrar y conservar accesibilidad; no justificarlo solo por suponer que cien filas son lentas.
- [ ] Mantener SVG y HTML nativos, traducciones y separación login/home. No externalizar recursos sensibles ni añadir una dependencia de frontend para sustituir unas pocas líneas.

**Dependencias:** 6 y 10. **Aceptación:** comparar presentación y comportamiento, además de líneas/bundle/DOM; si la mejora de tamaño añade más estados o regresiones, descartarla.

### 12. Cerrar documentación y validar el conjunto

- [ ] Actualizar `docs/browser.md`, `docs/routes.md`, `docs/sessions.md`, `docs/reverse-proxy.md` y ejemplos conforme a lo realmente implementado. Marcar revisiones históricas como tales: `docs/revision.md` contiene afirmaciones antiguas sobre secretos, correo y proxy que ya no describen el estado actual. Este plan no debe duplicar correcciones ya aplicadas.
- [ ] Ejecutar generación específica, pruebas existentes con `-race`, auditorías y la prueba multiinstancia con PostgreSQL/MySQL. Revalidar en navegador aislado cambio de BD con cookies antiguas, renovación simultánea, error SMTP/SQL, pérdida de conexión, cierre de sesión y actualización de assets. No usar sesiones reales del usuario.
- [ ] Mantener commits por problema: proxy, sesiones, cabeceras, privacidad, build, assets, servicios, SQL y frontend. Cada commit funcional debe incluir su documentación y comprobaciones relacionadas; evitar un commit final que mezcle correcciones de seguridad con cosmética.

No crear nuevas suites permanentes sin la autorización exigida por `AGENTS.md`; estos criterios pueden comprobarse con las pruebas existentes y diagnósticos temporales. Si falta cobertura para sostener un cambio delicado, pedir autorización específica para esa prueba al implementar. No se deben marcar tareas como completadas solo porque las comprobaciones antiguas pasen.

## Decisiones que no deben perderse durante la implementación

- La autenticación por cookies pertenece al adaptador web; la API sigue JSON/Bearer. No habilitar credentialed CORS como solución a problemas del formulario.
- CORS no impide que un atacante envíe peticiones fuera del navegador y no sustituye CSRF ni límites de abuso. `--dev` permisivo es intencional y no debe estar en producción.
- No cachear sesiones, permisos, OTP o revocación; no cambiar seguridad por menos consultas.
- No editar migraciones publicadas ni borrar sesiones de usuarios para simplificar una migración. El diseño de replay/retención debe acordarse antes de cambiar esquema.
- El [RFC 9700, sección 4.14](https://www.rfc-editor.org/rfc/rfc9700.html#section-4.14) sirve como referencia para protección de refresh; esta aplicación no se presenta como servidor OAuth completo. Distinguir rotación, detección de replay y caducidad absoluta.
- Sin CSS adicional, cambios de API ni nuevos endpoints por defecto. Menos líneas es un beneficio secundario, no una razón para quitar verificaciones, fallback o modularidad.
