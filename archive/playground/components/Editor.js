// Monaco Editor Wrapper Component

export class Editor {
    constructor(containerId, options = {}) {
        this.containerId = containerId;
        this.container = document.getElementById(containerId);
        this.editor = null;
        this.options = {
            value: '',
            language: 'javascript',
            theme: 'vs-dark',
            minimap: { enabled: false },
            fontSize: 14,
            lineNumbers: 'on',
            roundedSelection: false,
            scrollBeyondLastLine: false,
            readOnly: false,
            automaticLayout: true,
            ...options
        };

        this.changeListeners = [];
        this.init();
    }

    init() {
        if (!this.container) {
            throw new Error(`Container with id "${this.containerId}" not found`);
        }

        // Create Monaco editor instance
        this.editor = monaco.editor.create(this.container, this.options);

        // Setup change listener
        this.editor.onDidChangeModelContent(() => {
            this.notifyChangeListeners();
        });
    }

    getValue() {
        return this.editor ? this.editor.getValue() : '';
    }

    setValue(value) {
        if (this.editor) {
            this.editor.setValue(value || '');
        }
    }

    setLanguage(language) {
        if (this.editor) {
            const model = this.editor.getModel();
            if (model) {
                monaco.editor.setModelLanguage(model, language);
            }
        }
    }

    getLanguage() {
        if (this.editor) {
            const model = this.editor.getModel();
            if (model) {
                return model.getLanguageId();
            }
        }
        return null;
    }

    format() {
        if (this.editor) {
            this.editor.getAction('editor.action.formatDocument').run();
        }
    }

    getLineCount() {
        if (this.editor) {
            const model = this.editor.getModel();
            if (model) {
                return model.getLineCount();
            }
        }
        return 0;
    }

    insertAtCursor(text) {
        if (this.editor) {
            const selection = this.editor.getSelection();
            this.editor.executeEdits('', [{
                range: selection,
                text: text
            }]);
        }
    }

    getSelection() {
        if (this.editor) {
            const selection = this.editor.getSelection();
            const model = this.editor.getModel();
            if (model && selection) {
                return model.getValueInRange(selection);
            }
        }
        return '';
    }

    setReadOnly(readOnly) {
        if (this.editor) {
            this.editor.updateOptions({ readOnly });
        }
    }

    focus() {
        if (this.editor) {
            this.editor.focus();
        }
    }

    onChange(callback) {
        this.changeListeners.push(callback);
    }

    notifyChangeListeners() {
        const value = this.getValue();
        this.changeListeners.forEach(callback => {
            try {
                callback(value);
            } catch (error) {
                console.error('Error in change listener:', error);
            }
        });
    }

    dispose() {
        if (this.editor) {
            this.editor.dispose();
            this.editor = null;
        }
    }

    resize() {
        if (this.editor) {
            this.editor.layout();
        }
    }

    setTheme(theme) {
        if (this.editor) {
            monaco.editor.setTheme(theme);
        }
    }

    getEditor() {
        return this.editor;
    }
}
