"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.deactivate = exports.activate = void 0;
const vscode_1 = require("vscode");
const node_1 = require("vscode-languageclient/node");
let client;
function activate(context) {
    const serverCommand = 'wolf';
    const serverArgs = ['lsp'];
    const serverOptions = {
        run: { command: serverCommand, args: serverArgs },
        debug: { command: serverCommand, args: serverArgs }
    };
    const clientOptions = {
        documentSelector: [{ scheme: 'file', language: 'wolf' }],
        synchronize: {
            fileEvents: vscode_1.workspace.createFileSystemWatcher('**/*.wolf')
        }
    };
    client = new node_1.LanguageClient('wolfLanguageServer', 'Wolf Language Server', serverOptions, clientOptions);
    client.start();
}
exports.activate = activate;
function deactivate() {
    if (!client) {
        return undefined;
    }
    return client.stop();
}
exports.deactivate = deactivate;
//# sourceMappingURL=extension.js.map