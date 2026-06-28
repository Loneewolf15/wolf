"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.activate = activate;
exports.deactivate = deactivate;
const vscode = require("vscode");
const node_1 = require("vscode-languageclient/node");
let client;
function activate(context) {
    // Get path from settings or fallback to 'wolf' binary in PATH
    const config = vscode.workspace.getConfiguration('wolf');
    const wolfPath = config.get('path') || 'wolf';
    const serverOptions = {
        run: { command: wolfPath, args: ['lsp'] },
        debug: { command: wolfPath, args: ['lsp'] }
    };
    const clientOptions = {
        documentSelector: [{ scheme: 'file', language: 'wolf' }],
        synchronize: {
            fileEvents: vscode.workspace.createFileSystemWatcher('**/*.wolf')
        }
    };
    client = new node_1.LanguageClient('wolfLanguageServer', 'Wolf Language Server', serverOptions, clientOptions);
    client.start();
}
function deactivate() {
    if (!client) {
        return undefined;
    }
    return client.stop();
}
//# sourceMappingURL=extension.js.map