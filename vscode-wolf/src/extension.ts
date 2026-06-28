import * as vscode from 'vscode';
import {
  LanguageClient,
  LanguageClientOptions,
  ServerOptions
} from 'vscode-languageclient/node';

let client: LanguageClient;

export function activate(context: vscode.ExtensionContext) {
  // Get path from settings or fallback to 'wolf' binary in PATH
  const config = vscode.workspace.getConfiguration('wolf');
  const wolfPath = config.get<string>('path') || 'wolf';

  const serverOptions: ServerOptions = {
    run: { command: wolfPath, args: ['lsp'] },
    debug: { command: wolfPath, args: ['lsp'] }
  };

  const clientOptions: LanguageClientOptions = {
    documentSelector: [{ scheme: 'file', language: 'wolf' }],
    synchronize: {
      fileEvents: vscode.workspace.createFileSystemWatcher('**/*.wolf')
    }
  };

  client = new LanguageClient(
    'wolfLanguageServer',
    'Wolf Language Server',
    serverOptions,
    clientOptions
  );

  client.start();
}

export function deactivate(): Thenable<void> | undefined {
  if (!client) {
    return undefined;
  }
  return client.stop();
}
