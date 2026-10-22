# Cómo añadir una funcionalidad nueva

Este documento explica, paso a paso, cómo añadir una funcionalidad nueva a la API. Usa como ejemplo un módulo nuevo llamado **companies** (empresas), al lado de **tasks** (tareas). Todo lo que se describe ya existe en el proyecto para `task`, así que puedes compararlo con el código real mientras lees.

> Nota: esto es una guía, no código. Aquí no se añade nada. El ejemplo de `company` es ficticio y sirve para entender el proceso.

---

## 1. La idea general: de fuera hacia dentro

Una petición HTTP entra por fuera y baja por capas hasta la base de datos. Cada capa solo conoce a la de debajo a través de una interfaz:

```text
HTTP (contrato OpenAPI)      api/openapi.yaml + internal/adapter/httpapi
        │
Aplicación (casos de uso)    internal/company/application
        │
Dominio (reglas)             internal/company/domain        ← no depende de nada
        ▲
Infraestructura              internal/company/infrastructure/postgres
(implementa la interfaz del repositorio que define la aplicación)
```

Las dependencias van hacia dentro: la infraestructura depende de la aplicación, y la aplicación depende del dominio. El dominio no importa HTTP ni PostgreSQL. Esa es la regla que mantiene el código cambiable.

Y las piezas se conectan en un solo lugar: `cmd/api/main.go`. Ahí se crea cada implementación concreta y se pasa a quien la necesita.

---

## 2. Empieza por el contrato, no por el código

En este proyecto el contrato es `api/openapi.yaml`. Antes de escribir Go, describe la funcionalidad ahí.

Para companies, por ejemplo:

- `GET /companies` y `POST /companies`
- `GET /companies/{id}`, `PATCH /companies/{id}` y `DELETE /companies/{id}`
- Los esquemas `Company`, `CreateCompanyRequest` y `UpdateCompanyRequest`
- Las respuestas posibles de cada operación (`201`, `400`, `404`, `500`, etc.)
- `security: - bearerAuth: []` en las operaciones que requieren token

Después regeneras el código:

```sh
go generate ./internal/api
```

Esto crea en `internal/api/openapi.gen.go` los tipos Go y una interfaz `StrictServerInterface` con un método por cada operación nueva, por ejemplo `ListCompanies`, `CreateCompany`, etc.

**Por qué importa:** el compilador te dirá qué falta. Si el servidor no implementa un método de la interfaz, `go build` falla. El contrato te guía la lista de cosas a hacer.

---

## 3. Dominio: las reglas del negocio

Carpeta: `internal/company/domain/`

Aquí van la entidad y sus reglas, sin ninguna dependencia de HTTP ni de base de datos. Para tareas, esto es `internal/task/domain/task.go`.

Piensa en:

- **La entidad** (`Company` con `ID`, `Name`, `CreatedAt`).
- **Un constructor** (`NewCompany`) que valida y normaliza. Por ejemplo, quita espacios y rechaza un nombre vacío. Así una `Company` inválida no puede existir.
- **Errores de dominio** (`ErrCompanyNameRequired`, `ErrCompanyNotFound`). Son valores que el resto del código compara con `errors.Is`.
- **Tests** en `company_test.go`. Son tests puros, sin base de datos, y son los más rápidos del proyecto.

Lo que no va aquí: lógica de HTTP, consultas SQL, ni detalles de cómo se guarda nada.

---

## 4. Aplicación: los casos de uso

Carpeta: `internal/company/application/`

Hay dos cosas:

### a) La interfaz del repositorio

Un archivo `company_repository.go` define lo que la aplicación necesita, pero no cómo se implementa:

```go
type CompanyRepository interface {
    Create(ctx context.Context, company domain.Company) error
    Get(ctx context.Context, id string) (domain.Company, error)
    List(ctx context.Context) ([]domain.Company, error)
    Update(ctx context.Context, company domain.Company) error
    Delete(ctx context.Context, id string) error
}
```

Esta interfaz la define **quien la usa** (el servicio), no quien la implementa. En Go, las interfaces son implícitas: cualquier tipo que tenga esos métodos la cumple, sin declararlo. En TypeScript/NestJS sería parecido a un token de inyección, pero aquí no hay decoradores ni contenedor: el tipo cumple la interfaz por su forma.

### b) El servicio

`CompanyService` contiene los casos de uso: `CreateCompany`, `GetCompany`, `ListCompanies`, `UpdateCompany`, `DeleteCompany`. Cada uno:

1. Genera el ID si hace falta.
2. Construye la entidad con el constructor del dominio (así se validan las reglas).
3. Llama al repositorio.
4. Envuelve los errores con contexto (`fmt.Errorf("...: %w", err)`).

Mira `internal/task/application/task_service.go` como referencia.

---

## 5. Infraestructura: cómo se guarda

### a) La migración (la tabla)

Un archivo SQL nuevo en `internal/migration/sql/`, con el siguiente número libre:

```text
003_create_companies.sql
```

El nombre importa: el runner aplica los archivos en orden alfabético y registra en `schema_migrations` los que ya aplicó. **Nunca edites una migración ya aplicada**; crea una nueva.

Después de añadirla, ejecuta:

```sh
go run ./cmd/migrate
```

### b) El repositorio de PostgreSQL

Carpeta: `internal/company/infrastructure/postgres/`

Un tipo `CompanyRepository` que recibe un `*pgxpool.Pool` y escribe las consultas SQL. Debe cumplir la interfaz de la aplicación. Un truco útil que ya usa el proyecto:

```go
var _ application.CompanyRepository = (*CompanyRepository)(nil)
```

Si el tipo no cumple la interfaz, esa línea no compila. Es una comprobación gratuita.

Para tareas, el equivalente es `internal/task/infrastructure/postgres/task_repository.go`. Ahí también verás cómo se mapean los errores de PostgreSQL a errores de dominio.

### c) Un repositorio en memoria (opcional, para tests)

Igual que `internal/task/infrastructure/memory/`, puedes tener una implementación en memoria. Sirve para tests rápidos del servicio, sin base de datos.

---

## 6. Adaptador HTTP: traducir peticiones a casos de uso

Archivos: `internal/adapter/httpapi/`

Aquí implementas los métodos de la interfaz generada. La forma es la misma que en `openapi_server.go` para tareas:

- Recibir la petición ya decodificada (`request api.CreateCompanyRequestObject`).
- Llamar al servicio.
- Convertir el resultado a la respuesta del contrato, por ejemplo `api.CreateCompany201JSONResponse(...)`.
- Mapear errores de dominio a códigos HTTP con `errors.Is`: `ErrCompanyNotFound` → `404`, validación → `400`, lo demás → `500` con un mensaje genérico.

Dos detalles importantes:

1. **No expongas errores internos.** El cliente recibe `"internal server error"`. El detalle real va al log.
2. **Los métodos van en el mismo tipo `openAPIServer`.** Como vimos con la autenticación, en Go los métodos de un tipo pueden estar en varios archivos. Puedes crear `openapi_company.go` para mantenerlo ordenado.

Y el servidor necesita el servicio nuevo como campo, igual que `taskService` y `authenticationService` en `openapi_server.go`.

---

## 7. Composición: conectarlo todo

Archivo: `cmd/api/main.go`

Aquí se construye todo en orden y se pasa a quien lo necesita:

```go
companyService := company.NewService(databasePool)
apiHandler := httpapi.NewAPIHandler(taskService, authenticationService, companyService, readiness, logger)
```

Ahora la firma de `NewAPIHandler` crece con cada módulo. Cuando tengas tres o cuatro servicios, conviene agruparlos en un struct (por ejemplo `httpapi.Services`). Es una refactorización pequeña, y es mejor hacerla cuando aparezca el tercer módulo, no antes.

---

## 8. Protección: ¿necesita token?

Si la ruta necesita autenticación, hay que hacer tres cosas:

1. Marcar las operaciones con `security: - bearerAuth: []` en el spec. Esto es documentación y Swagger.
2. Protegerlas en el router, en `internal/adapter/httpapi/router.go`. Hoy solo `/tasks` y `/tasks/` están protegidas, con `RequireAccessToken`.
3. **Registrar las dos rutas:** `/companies` y `/companies/`. Es el mismo error que vimos con tareas: si solo registras `/companies`, las rutas con ID (`/companies/123`) caerían en el `/` general y no tendrían el middleware.

Como alternativa, puedes proteger todo el router de una vez y dejar públicas solo las rutas que lo necesiten. Es más seguro por defecto, pero requiere revisar cada ruta pública.

---

## 9. Pruebas: de abajo hacia arriba

Escribe las pruebas en el mismo orden que las capas. Así cada error se detecta en el nivel más bajo posible.

| Nivel | Qué pruebas | Dónde | Dependencias |
|---|---|---|---|
| Dominio | Validación y constructor | `domain/company_test.go` | Ninguna |
| Aplicación | Casos de uso con un repositorio falso | `application/company_service_test.go` | Repositorio en memoria |
| HTTP | Rutas, códigos, cuerpos, `401` sin token | `adapter/httpapi/company_http_test.go` | Servicio en memoria |
| Infraestructura | SQL real contra PostgreSQL | `infrastructure/postgres/..._integration_test.go` | Docker (tag `integration`) |

Para ejecutarlas:

```sh
go test ./...                                   # unitarias, sin Docker
TEST_DATABASE_URL='...' make integration        # integración, con la base de Docker
make ci                                         # lo que ejecuta la CI
```

Mira los tests de tareas para ver el patrón. Por ejemplo, `router_test.go` construye el router completo con `newTestRouter()`, y `task_http_test.go` prueba las rutas de tareas con él.

---

## 10. Verificación manual

Cuando las pruebas pasen, arranca la API y prueba con `curl` o con Swagger en `/docs`:

1. Regístrate y haz login para obtener un token.
2. Crea una empresa con el token.
3. Lístalas.
4. Pide una sin token y comprueba que devuelve `401`.
5. Pide una que no existe y comprueba que devuelve `404`.

Esto confirma que el contrato, el router, el servicio y la base de datos están conectados.

---

## 11. Documentación y lista de cambios

Antes de dar la funcionalidad por terminada:

- Actualiza la lista de endpoints en `README.md`.
- Actualiza `AGENTS.md` si cambia la arquitectura o la lista de rutas protegidas.
- Revisa `make ci` completo.

---

## Resumen: el orden

1. **Contrato:** `api/openapi.yaml`, luego `go generate ./internal/api`.
2. **Dominio:** entidad, constructor, errores, tests.
3. **Aplicación:** interfaz del repositorio, servicio, tests con repositorio falso.
4. **Infraestructura:** migración SQL, `go run ./cmd/migrate`, repositorio de PostgreSQL.
5. **Adaptador HTTP:** métodos sobre `openAPIServer`, mapeo de errores.
6. **Composición:** `cmd/api/main.go`.
7. **Protección:** seguridad en el spec y rutas en el router.
8. **Pruebas:** de abajo hacia arriba.
9. **Verificación:** `make ci`, `curl`, Swagger.
10. **Documentación:** README y AGENTS.md.

El compilador y las pruebas te dicen cuándo vas bien. Si algo no compila, casi siempre es porque falta un método del contrato, un campo en el constructor, o una ruta sin registrar.
