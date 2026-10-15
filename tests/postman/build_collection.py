"""Generates kanban.postman_collection.json.

The collection is kept as code because a hand-edited Postman export is hard
to review; run `python3 build_collection.py` after changing it.
"""
import json

items = []


def req(name, method, path, *, body=None, tests=(), pre=(), auth=True, headers=None):
    h = [{"key": "Content-Type", "value": "application/json"}] if body is not None else []
    if auth:
        h.append({"key": "Authorization", "value": "Bearer {{token}}"})
    for k, v in (headers or {}).items():
        h.append({"key": k, "value": v})
    item = {
        "name": name,
        "request": {
            "method": method,
            "header": h,
            "url": {"raw": "{{baseUrl}}" + path, "host": ["{{baseUrl}}"], "path": [p for p in path.split("?")[0].split("/") if p]},
        },
        "event": [],
    }
    if "?" in path:
        q = path.split("?", 1)[1]
        item["request"]["url"]["query"] = [{"key": kv.split("=")[0], "value": kv.split("=", 1)[1]} for kv in q.split("&")]
    if body is not None:
        item["request"]["body"] = {"mode": "raw", "raw": json.dumps(body, ensure_ascii=False, indent=2)}
    if pre:
        item["event"].append({"listen": "prerequest", "script": {"type": "text/javascript", "exec": list(pre)}})
    item["event"].append({"listen": "test", "script": {"type": "text/javascript", "exec": list(tests)}})
    items.append(item)


def status(code):
    return f"pm.test('status {code}', () => pm.response.to.have.status({code}));"


def code(c):
    return f"pm.test('error code {c}', () => pm.expect(pm.response.json().code).to.eql('{c}'));"


# The limit is 200 ms by default; a smoke run over the internet raises it with
# --env-var maxResponseMs=..., since network latency alone exceeds 200 ms there.
fast = ("const limit = Number(pm.variables.get('maxResponseMs') || 200); "
        "pm.test(`response under ${limit} ms`, () => pm.expect(pm.response.responseTime).to.be.below(limit));")

req("Readiness", "GET", "/readyz", auth=False, tests=[status(200), "pm.test('dependencies up', () => pm.expect(pm.response.json()).to.eql({postgres: 'up', redis: 'up'}));"])

req("TC-01 Register developer", "POST", "/api/v1/auth/register",
    pre=["pm.collectionVariables.set('devEmail', `dev-${Date.now()}@example.com`);"],
    body={"email": "{{devEmail}}", "name": "Newman Dev", "password": "password123"}, auth=False,
    tests=[status(201), "const u = pm.response.json();",
           "pm.test('developer role', () => pm.expect(u.role).to.eql('developer'));",
           "pm.test('no password in response', () => pm.expect(u).to.not.have.property('passwordHash'));",
           "pm.collectionVariables.set('devId', u.id);"])
req("TC-10 Register duplicate email", "POST", "/api/v1/auth/register",
    body={"email": "{{devEmail}}", "name": "Again", "password": "password123"}, auth=False,
    tests=[status(409), code("CONFLICT")])
req("Register invalid data", "POST", "/api/v1/auth/register",
    body={"email": "not-an-email", "name": "", "password": "short"}, auth=False,
    tests=[status(400), code("VALIDATION_ERROR"),
           "const fields = pm.response.json().details.map(d => d.field);",
           "pm.test('all invalid fields reported', () => pm.expect(fields).to.include.members(['email', 'name', 'password']));"])
req("TC-09 Login wrong password", "POST", "/api/v1/auth/login",
    body={"email": "{{adminEmail}}", "password": "wrong-password"}, auth=False,
    tests=[status(401), code("UNAUTHORIZED")])
req("Login admin", "POST", "/api/v1/auth/login",
    body={"email": "{{adminEmail}}", "password": "{{adminPassword}}"}, auth=False,
    # bcrypt is slow on purpose, so login is exempt from the 200 ms check
    tests=[status(200), "const r = pm.response.json();",
           "pm.test('access token issued', () => pm.expect(r.accessToken).to.be.a('string').and.not.empty);",
           "pm.test('refresh cookie is HttpOnly', () => pm.expect(pm.response.headers.all().filter(h => h.key.toLowerCase() === 'set-cookie').map(h => h.value).join(';')).to.include('HttpOnly'));",
           "pm.collectionVariables.set('token', r.accessToken);"])
req("Me", "GET", "/api/v1/auth/me", tests=[status(200), "pm.test('admin role', () => pm.expect(pm.response.json().role).to.eql('admin'));"])
req("TC-13 Projects without token", "GET", "/api/v1/projects", auth=False, tests=[status(401), code("UNAUTHORIZED")])

req("TC-14 Create project", "POST", "/api/v1/projects",
    pre=["const letters = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ';",
         "let key = 'N'; for (let i = 0; i < 7; i++) key += letters[Math.floor(Math.random() * 26)];",
         "pm.collectionVariables.set('projectKey', key);"],
    body={"key": "{{projectKey}}", "name": "Newman project", "description": "API tests"},
    tests=[status(201), fast, "pm.collectionVariables.set('projectId', pm.response.json().id);"])
req("Create project with duplicate key", "POST", "/api/v1/projects",
    body={"key": "{{projectKey}}", "name": "Duplicate"}, tests=[status(409), code("CONFLICT")])
req("Create project with unknown field", "POST", "/api/v1/projects",
    body={"key": "ZZZ", "name": "x", "owner": "me"}, tests=[status(400), code("VALIDATION_ERROR")])
req("TC-17 Create board", "POST", "/api/v1/projects/{{projectId}}/boards", body={"name": "Sprint"},
    tests=[status(201), "const b = pm.response.json();",
           "pm.test('default columns', () => pm.expect(b.columns.map(c => c.name)).to.eql(['To Do', 'In Progress', 'Done']));",
           "pm.collectionVariables.set('boardId', b.id);",
           "pm.collectionVariables.set('todoId', b.columns[0].id);",
           "pm.collectionVariables.set('doingId', b.columns[1].id);"])
req("Set WIP limit", "PATCH", "/api/v1/columns/{{doingId}}", body={"wipLimit": 1},
    tests=[status(200), "pm.test('limit stored', () => pm.expect(pm.response.json().wipLimit).to.eql(1));"])
req("Create label", "POST", "/api/v1/projects/{{projectId}}/labels", body={"name": "bug", "color": "#FF0000"},
    tests=[status(201), "pm.collectionVariables.set('labelId', pm.response.json().id);",
           "pm.test('colour normalised', () => pm.expect(pm.response.json().color).to.eql('#ff0000'));"])
req("TC-02 Create card", "POST", "/api/v1/columns/{{todoId}}/cards",
    body={"title": "Login form", "priority": "high", "labelIds": ["{{labelId}}"], "dueDate": "2027-01-31"},
    tests=[status(201), fast, "const c = pm.response.json();",
           "pm.test('fields stored', () => { pm.expect(c.priority).to.eql('high'); pm.expect(c.dueDate).to.eql('2027-01-31'); pm.expect(c.labelIds).to.eql([pm.collectionVariables.get('labelId')]); });",
           "pm.collectionVariables.set('card1', c.id);"])
req("TC-05 Create card without title", "POST", "/api/v1/columns/{{todoId}}/cards", body={"title": "  "},
    tests=[status(400), code("VALIDATION_ERROR"),
           "pm.test('title message', () => pm.expect(pm.response.json().details[0]).to.eql({field: 'title', message: 'Название обязательно'}));"])
req("Create second card", "POST", "/api/v1/columns/{{todoId}}/cards", body={"title": "Reports", "description": "CSV export"},
    tests=[status(201), "pm.collectionVariables.set('card2', pm.response.json().id);"])
req("TC-03 Move card", "POST", "/api/v1/cards/{{card1}}/move", body={"columnId": "{{doingId}}"},
    tests=[status(200), "pm.test('column changed', () => pm.expect(pm.response.json().columnId).to.eql(pm.collectionVariables.get('doingId')));"])
req("TC-07 Move card into full column", "POST", "/api/v1/cards/{{card2}}/move", body={"columnId": "{{doingId}}"},
    tests=[status(409), code("WIP_LIMIT_EXCEEDED")])
req("TC-20 Update card", "PATCH", "/api/v1/cards/{{card1}}", body={"title": "Login page", "assigneeId": None, "labelIds": []},
    tests=[status(200), "const c = pm.response.json();",
           "pm.test('updated', () => { pm.expect(c.title).to.eql('Login page'); pm.expect(c.labelIds).to.eql([]); });"])
req("TC-06 Search", "GET", "/api/v1/boards/{{boardId}}/cards?q=csv",
    tests=[status(200), fast, "const r = pm.response.json();",
           "pm.test('one match', () => { pm.expect(r.total).to.eql(1); pm.expect(r.items[0].title).to.eql('Reports'); });"])
req("TC-26 Search with SQL injection", "GET", "/api/v1/boards/{{boardId}}/cards?q=' OR 1=1 --",
    tests=[status(200), "pm.test('nothing matched', () => pm.expect(pm.response.json().total).to.eql(0));"])
req("Filter by invalid priority", "GET", "/api/v1/boards/{{boardId}}/cards?priority=urgent",
    tests=[status(400), code("VALIDATION_ERROR")])
req("Activity", "GET", "/api/v1/cards/{{card1}}/activity",
    tests=[status(200), "const actions = pm.response.json().map(a => a.action);",
           "pm.test('history recorded', () => pm.expect(actions).to.include.members(['created', 'moved', 'updated', 'labels_changed']));"])
req("TC-21 Comment", "POST", "/api/v1/cards/{{card1}}/comments", body={"body": "Looks good"}, tests=[status(201)])
req("TC-19 Delete non-empty column", "DELETE", "/api/v1/columns/{{todoId}}", tests=[status(409), code("COLUMN_NOT_EMPTY")])

req("Login developer", "POST", "/api/v1/auth/login",
    body={"email": "{{devEmail}}", "password": "password123"}, auth=False,
    tests=[status(200), "pm.collectionVariables.set('devToken', pm.response.json().accessToken);"])
req("TC-16 Developer reads foreign board", "GET", "/api/v1/boards/{{boardId}}", auth=False,
    headers={"Authorization": "Bearer {{devToken}}"}, tests=[status(403), code("FORBIDDEN")])
req("TC-04 Developer deletes project", "DELETE", "/api/v1/projects/{{projectId}}", auth=False,
    headers={"Authorization": "Bearer {{devToken}}"}, tests=[status(403), code("FORBIDDEN")])
req("Developer creates project", "POST", "/api/v1/projects", auth=False,
    headers={"Authorization": "Bearer {{devToken}}"}, body={"key": "DEVX", "name": "x"}, tests=[status(403)])

req("Delete card", "DELETE", "/api/v1/cards/{{card2}}", tests=[status(204)])
req("Deleted card is gone", "GET", "/api/v1/cards/{{card2}}", tests=[status(404), code("NOT_FOUND")])
req("Delete project", "DELETE", "/api/v1/projects/{{projectId}}", tests=[status(204)])

collection = {
    "info": {
        "name": "Kanban Lab API",
        "description": "Сценарные API-тесты. Запуск: newman run kanban.postman_collection.json -e local.postman_environment.json",
        "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
    },
    "item": items,
    "variable": [{"key": k, "value": ""} for k in
                 ["token", "devEmail", "devId", "devToken", "projectKey", "projectId", "boardId", "todoId",
                  "doingId", "labelId", "card1", "card2"]],
}
with open("kanban.postman_collection.json", "w", encoding="utf-8") as f:
    json.dump(collection, f, ensure_ascii=False, indent=2)
    f.write("\n")
print(f"{len(items)} requests")
