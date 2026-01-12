import * as vscode from 'vscode';

export class StatusBarManager {
    private statusBarItem: vscode.StatusBarItem;
    private issueCount: number = 0;
    private isAnalyzing: boolean = false;

    constructor() {
        this.statusBarItem = vscode.window.createStatusBarItem(
            vscode.StatusBarAlignment.Left,
            100
        );
        this.statusBarItem.command = 'unqueryvet.showOutput';
        this.update();
        this.statusBarItem.show();
    }

    public setAnalyzing(analyzing: boolean): void {
        this.isAnalyzing = analyzing;
        this.update();
    }

    public setIssueCount(count: number): void {
        this.issueCount = count;
        this.isAnalyzing = false;
        this.update();
    }

    public setError(message: string): void {
        this.statusBarItem.text = `$(error) Unqueryvet: ${message}`;
        this.statusBarItem.backgroundColor = new vscode.ThemeColor('statusBarItem.errorBackground');
        this.statusBarItem.tooltip = message;
    }

    public setDisabled(): void {
        this.statusBarItem.text = '$(circle-slash) Unqueryvet: Disabled';
        this.statusBarItem.backgroundColor = undefined;
        this.statusBarItem.tooltip = 'Click to enable';
    }

    private update(): void {
        if (this.isAnalyzing) {
            this.statusBarItem.text = '$(sync~spin) Unqueryvet: Analyzing...';
            this.statusBarItem.backgroundColor = undefined;
            this.statusBarItem.tooltip = 'Analysis in progress';
        } else if (this.issueCount === 0) {
            this.statusBarItem.text = '$(check) Unqueryvet: OK';
            this.statusBarItem.backgroundColor = undefined;
            this.statusBarItem.tooltip = 'No SELECT * issues found';
        } else {
            this.statusBarItem.text = `$(warning) Unqueryvet: ${this.issueCount} issue${this.issueCount !== 1 ? 's' : ''}`;
            this.statusBarItem.backgroundColor = new vscode.ThemeColor('statusBarItem.warningBackground');
            this.statusBarItem.tooltip = `${this.issueCount} SELECT * issue${this.issueCount !== 1 ? 's' : ''} found. Click to view.`;
        }
    }

    public dispose(): void {
        this.statusBarItem.dispose();
    }
}
