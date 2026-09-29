const ownerInput = document.getElementById("owner-id");
const statusLine = document.getElementById("status");
const filesBody = document.getElementById("files");

ownerInput.value = localStorage.getItem("garchive-owner") || "user-1";

document.getElementById("owner-form").addEventListener("submit", (event) => {
  event.preventDefault();
  localStorage.setItem("garchive-owner", ownerId());
  loadFiles();
});

document.getElementById("upload-form").addEventListener("submit", (event) => {
  event.preventDefault();
  uploadFile();
});

loadFiles();

function ownerId() {
  return ownerInput.value.trim();
}

function setStatus(text, isError) {
  statusLine.textContent = text;
  statusLine.classList.toggle("error", Boolean(isError));
}

async function loadFiles() {
  const owner = ownerId();
  if (!owner) {
    setStatus("Укажите owner id", true);
    return;
  }

  const response = await fetch("/api/files?owner_id=" + encodeURIComponent(owner));
  const payload = await readJSON(response);
  if (!response.ok) {
    setStatus(payload.error || "Не удалось получить список", true);
    render([]);
    return false;
  }

  render(payload.files || []);
  return true;
}

async function uploadFile() {
  const input = document.getElementById("file");
  const file = input.files && input.files[0];
  if (!file) {
    setStatus("Выберите файл", true);
    return;
  }

  const body = new FormData();
  body.set("owner_id", ownerId());
  body.set("file", file);

  setStatus("Загрузка файла…", false);
  const response = await fetch("/api/files", { method: "POST", body });
  const payload = await readJSON(response);
  if (!response.ok) {
    setStatus(payload.error || "Не удалось загрузить файл", true);
    return;
  }

  input.value = "";
  if (await loadFiles()) {
    setStatus("Файл загружен", false);
  }
}

async function deleteFile(id) {
  const response = await fetch(
    "/api/files/" + encodeURIComponent(id) + "?owner_id=" + encodeURIComponent(ownerId()),
    { method: "DELETE" },
  );
  if (!response.ok && response.status !== 204) {
    const payload = await readJSON(response);
    setStatus(payload.error || "Не удалось удалить файл", true);
    return;
  }
  if (await loadFiles()) {
    setStatus("Файл удалён", false);
  }
}

function render(files) {
  filesBody.replaceChildren();
  if (files.length === 0) {
    const row = document.createElement("tr");
    const cell = document.createElement("td");
    cell.colSpan = 5;
    cell.className = "empty";
    cell.textContent = "Файлов нет";
    row.append(cell);
    filesBody.append(row);
    return;
  }

  for (const file of files) {
    const row = document.createElement("tr");

    const name = document.createElement("td");
    name.textContent = file.original_name;

    const size = document.createElement("td");
    size.textContent = formatSize(file.size_bytes);

    const status = document.createElement("td");
    status.textContent = file.status;

    const created = document.createElement("td");
    created.textContent = formatTime(file.created_at);

    const actions = document.createElement("td");
    actions.className = "actions";
    if (file.status === "ready") {
      const download = document.createElement("a");
      download.href = "/api/files/" + encodeURIComponent(file.id) + "/content?owner_id=" + encodeURIComponent(ownerId());
      download.textContent = "Скачать";
      actions.append(download);
    }

    const remove = document.createElement("button");
    remove.type = "button";
    remove.textContent = "Удалить";
    remove.addEventListener("click", () => deleteFile(file.id));
    actions.append(remove);

    row.append(name, size, status, created, actions);
    filesBody.append(row);
  }
}

async function readJSON(response) {
  if (response.status === 204) {
    return {};
  }
  try {
    return await response.json();
  } catch {
    return {};
  }
}

function formatSize(bytes) {
  const size = Number(bytes) || 0;
  if (size < 1024) {
    return size + " Б";
  }
  if (size < 1024 * 1024) {
    return (size / 1024).toFixed(1) + " КБ";
  }
  return (size / (1024 * 1024)).toFixed(1) + " МБ";
}

function formatTime(value) {
  if (!value) {
    return "";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toLocaleString("ru-RU");
}
