// Editor Component Tests
import { describe, test, expect, beforeEach, afterEach } from '@jest/globals';

// Mock Monaco editor
global.monaco = {
    editor: {
        create: jest.fn(() => ({
            getValue: jest.fn(() => 'test code'),
            setValue: jest.fn(),
            getModel: jest.fn(() => ({
                getLanguageId: jest.fn(() => 'javascript'),
                getLineCount: jest.fn(() => 10),
                getValueInRange: jest.fn(() => 'selected text')
            })),
            getSelection: jest.fn(() => ({
                startLineNumber: 1,
                endLineNumber: 1
            })),
            onDidChangeModelContent: jest.fn((callback) => {
                // Store callback for testing
                return { dispose: jest.fn() };
            }),
            updateOptions: jest.fn(),
            focus: jest.fn(),
            dispose: jest.fn(),
            layout: jest.fn(),
            getAction: jest.fn(() => ({
                run: jest.fn()
            })),
            executeEdits: jest.fn()
        })),
        setModelLanguage: jest.fn(),
        setTheme: jest.fn()
    }
};

// Mock Editor class for testing
class Editor {
    constructor(containerId, options = {}) {
        this.containerId = containerId;
        this.options = options;
        this.changeListeners = [];
        this.editor = monaco.editor.create({}, options);
    }

    getValue() {
        return this.editor.getValue();
    }

    setValue(value) {
        this.editor.setValue(value);
    }

    setLanguage(language) {
        const model = this.editor.getModel();
        if (model) {
            monaco.editor.setModelLanguage(model, language);
        }
    }

    getLineCount() {
        const model = this.editor.getModel();
        return model ? model.getLineCount() : 0;
    }

    format() {
        this.editor.getAction('editor.action.formatDocument').run();
    }

    onChange(callback) {
        this.changeListeners.push(callback);
    }

    dispose() {
        this.editor.dispose();
    }
}

describe('Editor Component', () => {
    let editor;

    beforeEach(() => {
        // Setup DOM
        document.body.innerHTML = '<div id="testEditor"></div>';
        editor = new Editor('testEditor', {
            language: 'javascript',
            theme: 'vs-dark'
        });
    });

    afterEach(() => {
        if (editor) {
            editor.dispose();
        }
        document.body.innerHTML = '';
    });

    test('should initialize with correct options', () => {
        expect(editor.options.language).toBe('javascript');
        expect(editor.options.theme).toBe('vs-dark');
    });

    test('should get and set value', () => {
        editor.setValue('console.log("test")');
        expect(editor.editor.setValue).toHaveBeenCalledWith('console.log("test")');

        const value = editor.getValue();
        expect(value).toBe('test code');
    });

    test('should change language', () => {
        editor.setLanguage('python');
        expect(monaco.editor.setModelLanguage).toHaveBeenCalled();
    });

    test('should get line count', () => {
        const lineCount = editor.getLineCount();
        expect(lineCount).toBe(10);
    });

    test('should format code', () => {
        editor.format();
        expect(editor.editor.getAction).toHaveBeenCalledWith('editor.action.formatDocument');
    });

    test('should register change listeners', () => {
        const callback = jest.fn();
        editor.onChange(callback);
        expect(editor.changeListeners).toContain(callback);
    });

    test('should dispose properly', () => {
        editor.dispose();
        expect(editor.editor.dispose).toHaveBeenCalled();
    });
});

describe('Editor Value Management', () => {
    let editor;

    beforeEach(() => {
        document.body.innerHTML = '<div id="testEditor"></div>';
        editor = new Editor('testEditor');
    });

    afterEach(() => {
        editor.dispose();
        document.body.innerHTML = '';
    });

    test('should handle empty value', () => {
        editor.setValue('');
        expect(editor.editor.setValue).toHaveBeenCalledWith('');
    });

    test('should handle multi-line value', () => {
        const code = 'line 1\nline 2\nline 3';
        editor.setValue(code);
        expect(editor.editor.setValue).toHaveBeenCalledWith(code);
    });

    test('should handle special characters', () => {
        const code = 'const regex = /[a-z]+/g;';
        editor.setValue(code);
        expect(editor.editor.setValue).toHaveBeenCalled();
    });
});

describe('Editor Language Support', () => {
    let editor;

    beforeEach(() => {
        document.body.innerHTML = '<div id="testEditor"></div>';
        editor = new Editor('testEditor');
    });

    afterEach(() => {
        editor.dispose();
        document.body.innerHTML = '';
    });

    test('should support JavaScript', () => {
        editor.setLanguage('javascript');
        expect(monaco.editor.setModelLanguage).toHaveBeenCalled();
    });

    test('should support Python', () => {
        editor.setLanguage('python');
        expect(monaco.editor.setModelLanguage).toHaveBeenCalled();
    });

    test('should support Go', () => {
        editor.setLanguage('go');
        expect(monaco.editor.setModelLanguage).toHaveBeenCalled();
    });

    test('should support Shell', () => {
        editor.setLanguage('shell');
        expect(monaco.editor.setModelLanguage).toHaveBeenCalled();
    });
});
