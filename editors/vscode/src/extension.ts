import * as path from 'path';
import { workspace, ExtensionContext } from 'vscode';

import {
	LanguageClient,
	LanguageClientOptions,
	ServerOptions
} from 'vscode-languageclient/node';

let client: LanguageClient;

export function activate(context: ExtensionContext) {
	const serverCommand = 'wolf';
	const serverArgs = ['lsp'];

	const serverOptions: ServerOptions = {
		run: { command: serverCommand, args: serverArgs },
		debug: { command: serverCommand, args: serverArgs }
	};

	const clientOptions: LanguageClientOptions = {
		documentSelector: [{ scheme: 'file', language: 'wolf' }],
		synchronize: {
			fileEvents: workspace.createFileSystemWatcher('**/*.wolf')
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
