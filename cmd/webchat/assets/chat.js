
function newElement(tag, classes, child) {
    const elem = document.createElement(tag);
    if (classes) {
        elem.setAttribute("class", classes);
    }
    if (child) {
        elem.appendChild(child);
    }
    return elem;
}

function scrollToEnd() {
    window.scrollTo(0, document.body.scrollHeight);
}

function selectLink(list, id) {
    showConfigForm(false);
    for (const el of list.querySelectorAll("a")) {
        const item = el.getAttribute("id");
        el.setAttribute("class", "pure-menu-link" + ((item === id) ? " selected-link" : ""));
    }
}

function refreshChatList(list, currentID) {
    console.log("refresh chat list: current=%s", currentID);
    const parent = document.getElementById("conv-list");
    parent.replaceChildren();
    if (list) {
        for (const item of list) {
            const link = newElement("a", "pure-menu-link");
            link.setAttribute("id", item.id);
            link.textContent = item.summary;
            const entry = newElement("li", "pure-menu-item", link);
            parent.appendChild(entry);
        }
        if (currentID) {
            selectLink(parent, currentID);
        }
    }
}

function addMessage(chat, msg, showReasoning) {
    if (msg.reasoning && msg.reasoning.trim()) {
        if (!msg.update) {
            extendMessageList(chat, msg.role, true, showReasoning, msg.excluded);
        }
        addContent(chat, msg.reasoning);
    }
    if (msg.content && msg.content.trim()) {
        if (!msg.update) {
            extendMessageList(chat, msg.role, false, showReasoning, msg.excluded);
        }
        addContent(chat, msg.content);
    }
}

function extendMessageList(chat, role, isReasoning, showReasoning, excluded) {
    var type = "final";
    var skip = false;
    if (role === "user") {
        type = "user";
    } else if (role === "tool" || isReasoning) {
        type = "analysis";
        const n = chat.querySelectorAll("li");
        skip = (n.length > 0) && n[n.length-1].classList.contains("analysis");
    }
    if (skip) {
        // add a new message part to the last message
        const list = chat.getElementsByClassName("msg");
        if (list.length === 0) {
            console.error("chat update error: empty message list");
            return;
        }
        list[list.length-1].appendChild(newElement("div", "msgpart"));
    } else {
        // add a new message to the list
        var className = "msg";
        if (excluded) {
            className += " excluded";
        }
        const entry = newElement("li", "chat-item "+type, newElement("div", className, newElement("div", "msgpart")));
        if (type === "analysis" && !showReasoning) {
            entry.style.display = "none";
        }
        chat.appendChild(entry);
    }
}

function addContent(chat, content) {
    const nodes = chat.querySelectorAll("div.msgpart");
    if (nodes.length === 0) {
        console.error("chat update error: empty node list");
        return;
    }
    nodes[nodes.length - 1].innerHTML = content;
    scrollToEnd();
}

function loadChat(chat, conv, showReasoning) {
    console.log("load chat %s reasoning=%s", conv.id, showReasoning);
    chat.replaceChildren();
    if (conv.messages) {
        for (const msg of conv.messages) {
            addMessage(chat, msg, showReasoning);
        }
    }
    const list = document.getElementById("conv-list");
    selectLink(list, conv.id);
}

function refreshChat(chat, showReasoning) {
    console.log("refresh chat reasoning=%s", showReasoning);
    for (const item of chat.querySelectorAll("li.analysis")) {
        item.style.display = (showReasoning) ? "flex" : "none";
    }
}

function showConfigForm(on) {
    document.getElementById("chat-list").style.display = (on) ? "none" : "block";
    document.getElementById("input-box").style.display = (on) ? "none" : "block";
    document.getElementById("config-page").style.display = (on) ? "block" : "none"; 
}


function setConfig(cfg) {
    console.log("setConfig", cfg);

    const id = cfg.model;
    document.getElementById("model-name").textContent = id;

    const form = document.getElementById("config-form");
    const modelSelect = document.getElementById("model-select");
    var options = "";
    for (const name in cfg.models) {
        const selected = (name === id) ? " selected" : "";
        options += `<option${selected}>${name}</option>`
    }
    modelSelect.innerHTML = options;
    form.system.value = cfg.system_prompt;
    setGenerationConfig(form, cfg.models[id]);

    const parent = document.getElementById("tools-list");
    parent.replaceChildren();
    if (cfg.tools) {
        for (const tool of cfg.tools) {
            const checkbox = newElement("div", "tool-checkbox");
            const checked = (tool.enabled) ? "checked" : "";
            checkbox.innerHTML = `<input id="${tool.name}-tool" name="${tool.name}_tool" type="checkbox" ${checked}> <label for="${tool.name}-tool">${tool.name}</label><br>`;
            parent.appendChild(checkbox);
        }
    }
}

function setGenerationConfig(form, values) {
    form.temperature.value = values.temperature || "";
    form.top_p.value = values.top_p || "";
    form.top_k.value = values.top_k || "";
    form.presence_penalty.value = values.presence_penalty || "";
    form.repetition_penalty.value = values.repetition_penalty || "";    
    const radio = form.querySelectorAll(`input[name="reasoning"]`);
    for (const el of radio) {
        el.checked = (el.value === values.reasoning_effort);
    }
}

function clearStats() {
    for (const fld of ["context-size", "tokens-generated", "prompt-speed", "stats-speed", "stats-tools"]) {
        document.getElementById(fld).textContent = "";
    }
}

function updateStats(stats) {
    document.getElementById("context-size").textContent = `context: ${stats.context_size}`;
    document.getElementById("tokens-generated").textContent = `generated: ${stats.tokens_generated}`;
    if (stats.prompt_speed && stats.generation_speed) {
        document.getElementById("prompt-speed").textContent = `pp: ${stats.prompt_speed.toFixed(1)} tps`;
        document.getElementById("stats-speed").textContent = `tg: ${stats.generation_speed.toFixed(1)} tps`;
    } else {
        document.getElementById("prompt-speed").textContent = "";
        document.getElementById("stats-speed").textContent = `in ${duration(stats.generation_time)}`;
    }
    if (stats.tool_calls) {
        document.getElementById("stats-tools").textContent = `${stats.tool_calls} tool calls in ${duration(stats.tool_time)}`;
    } else {
        document.getElementById("stats-tools").textContent = "";
    }
}

function duration(ms) {
    return (ms >= 1000) ? (ms/1000).toFixed(1)+"s" : parseInt(ms)+"ms";
}

function initFormControls(app) {
    const form = document.getElementById("config-form");
    const select = document.getElementById("model-select");

    select.addEventListener("change", e => {
        submitConfigForm(app, form);
    })
    form.addEventListener("submit", e => {
        e.preventDefault();
        submitConfigForm(app, form, true);

    })
}

function submitConfigForm(app, form, withGenerationConfig) {
    var cfg = {
        model: form.model.value,
        system_prompt: form.system.value,
        models: {},
        tools: []
    };
    const tools = form.querySelectorAll(`.tool-checkbox input`);
    for (const el of tools) {
        cfg.tools.push({ name: el.name.slice(0, -5), enabled: el.checked });    
    }
    if (withGenerationConfig) {
        var reasoning_effort = "medium";
        const radio = form.querySelectorAll(`input[name="reasoning"]`);
        for (const el of radio) {
            if (el.checked) reasoning_effort = el.value;
        }
        cfg.models[form.model.value] = {
            temperature: parseFloat(form.temperature.value),
            top_p: parseFloat(form.top_p.value),
            top_k: parseInt(form.top_k.value),
            presence_penalty: parseFloat(form.presence_penalty.value),
            repetition_penalty: parseFloat(form.repetition_penalty.value),
            reasoning_effort: reasoning_effort
        }
    }
    console.log("update config", cfg);
    app.send({ type: "config", config: cfg });    
}


function initMenuControls(app) {
    const list = document.getElementById("conv-list");

    list.addEventListener("click", e => {
        const link = e.target.closest("a");
        if (link) {
            const id = link.getAttribute("id");
            selectLink(list, id);
            app.currentChatID = id;
            app.send({ type: "load", id: id });
        }
    });

    document.getElementById("new-chat").addEventListener("click", e => {
        selectLink(list, "");
        app.currentChatID = "";
        app.send({ type: "load" });
    });

    document.getElementById("del-chat").addEventListener("click", e => {
        showConfigForm(false);
        for (const el of list.querySelectorAll("a")) {
            if (el.getAttribute("class").includes("selected-link")) {
                app.send({ type: "delete", id: el.getAttribute("id") });
                app.currentChatID = "";
                break;
            }
        }
    });

    document.getElementById("options").addEventListener("click", e => {
        app.send({ type: "config" });
        showConfigForm(true);
    });

    const checkbox = document.getElementById("reasoning-history");
    checkbox.addEventListener("click", e => {
        app.showReasoning = checkbox.checked;
        refreshChat(app.chat, app.showReasoning);
    });
}

function initChatControls(app) {
    app.chat.addEventListener("click", e => {
        const collapsed = e.target.closest(".tool-response");
        if (collapsed) {
            collapsed.setAttribute("class", "tool-response-expanded");
            return;
        }
        const expanded = e.target.closest(".tool-response-expanded");
        if (expanded) {
            expanded.setAttribute("class", "tool-response");
        }
    });
}

function initInputTextbox(app) {
    const input = document.getElementById("input-text");

    const submit = function () {
        console.log("send add message");
        showConfigForm(false);
        const msg = input.value;
        if (msg.trim() === "") {
            input.placeholder = "Please enter a question";
            input.value = "";
            return;
        }
        input.value = "";
        input.setAttribute("class", "input-default");

        if (!app.showReasoning) {
            refreshChat(app.chat, false);
        }
        addMessage(app.chat, {role: "user", content: `<p>${msg}</p>`});
        app.send({ type: "chat", message: { role: "user", content: msg } });
        input.placeholder = "Type a message (Shift+Enter to add a new line)";
    }

    input.addEventListener("keypress", e => {
        if (e.key === "Enter") {
            if (e.shiftKey) {
                input.setAttribute("class", "input-expanded");
            } else {
                e.preventDefault();
                submit();
            }
        }
    });
    input.addEventListener("blur", e => {
        input.setAttribute("class", "input-default");
    });
    document.getElementById("send-button").addEventListener("click", submit);
}

// Websocket communication with server
class App {
    constructor() {
        this.ws = null;
        this.connected = false;
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = 10;
        this.reconnectDelay = 1000;
        this.heartbeatInterval = null;
        this.pongTimeout = null;
        this.chat = document.getElementById("chat-list");
        this.showReasoning = document.getElementById("reasoning-history").checked;
        this.currentChatID = "";
        initInputTextbox(this);
        initMenuControls(this);
        initFormControls(this);
        initChatControls(this);
    }

    connect() {
        this.ws = new WebSocket("/websocket");
        
        this.ws.addEventListener("open", e => {
            console.log("websocket connected");
            this.connected = true;
            document.getElementById("error-message").textContent = "";
            this.reconnectAttempts = 0;
            this.reconnectDelay = 1000;

            this.heartbeatInterval = setInterval(() => {
                this.send({ type: "ping" });
                this.pongTimeout = setTimeout(() => {
                    console.log("no pong received - closing connection");
                    this.ws.close(1011, "no pong response");
                }, 10000);
            }, 30000);
            
            this.init();
        });
        
        this.ws.addEventListener("close", e => {
            console.log(`websocket closed - code=${e.code} reason=${e.reason}`);
            this.connected = false;
            clearStats();
            document.getElementById("error-message").textContent = "Server disconnected";
            clearInterval(this.heartbeatInterval);
            clearTimeout(this.pongTimeout);
            this.reconnect();
        });
        
        this.ws.addEventListener("error", error => {
            console.error("websocket error:", error);
            this.ws.close();
        });

        this.ws.addEventListener("message", e => {
            const resp = JSON.parse(e.data);
            this.recv(resp);
        });  
    }

    reconnect() {
        if (this.reconnectAttempts >= this.maxReconnectAttempts) {
            console.error("max reconnection attempts reached - stopping");
            return;
        }
        this.reconnectAttempts++;
        console.log(`websocket reconnect attempt ${this.reconnectAttempts}...`);
        setTimeout(() => this.connect(), this.reconnectDelay);
        this.reconnectDelay *= 2;
    }

    init() {
        this.send({ type: "list" });
        if (this.currentChatID) {
            this.send({ type: "load", id: this.currentChatID });
        }
    }

    recv(resp) {
        console.log("recv", resp.type);
        switch (resp.type) {
            case "chat":
                addMessage(this.chat, resp.message, true);
                if (resp.message.end && !this.showReasoning) {
                    refreshChat(app.chat, false);
                }
                break;
            case "stats":
                updateStats(resp.stats);
                break;
            case "list":
                refreshChatList(resp.list, resp.current_id);
                break;
            case "load":
                loadChat(this.chat, resp.conversation, this.showReasoning);
                break;
            case "config":
                setConfig(resp.config);
                break;
            case "pong":
                clearTimeout(this.pongTimeout);
                break;
            default:
                console.error("received message with unknown type: ", resp.type)
        }
    }

    send(req) {
        if (this.connected) {
            console.log("send", req.type);
            this.ws.send(JSON.stringify(req));
        } else {
            console.error("send %s failed - not connected", req.type);
        }
    }
}

const app = new App();
app.connect();


