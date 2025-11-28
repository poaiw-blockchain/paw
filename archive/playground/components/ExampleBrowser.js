// Example Browser Component

export class ExampleBrowser {
    constructor(containerId, examples) {
        this.containerId = containerId;
        this.container = document.getElementById(containerId);
        this.examples = examples;
        this.filteredExamples = examples;
        this.selectedExample = null;

        if (!this.container) {
            throw new Error(`Container with id "${containerId}" not found`);
        }
    }

    filter(query) {
        const lowerQuery = query.toLowerCase();
        this.filteredExamples = Object.entries(this.examples).reduce((acc, [key, example]) => {
            const title = (example.title || '').toLowerCase();
            const description = (example.description || '').toLowerCase();
            const category = (example.category || '').toLowerCase();

            if (title.includes(lowerQuery) || description.includes(lowerQuery) || category.includes(lowerQuery)) {
                acc[key] = example;
            }

            return acc;
        }, {});

        return this.filteredExamples;
    }

    selectExample(exampleId) {
        this.selectedExample = exampleId;
        return this.examples[exampleId] || null;
    }

    getExample(exampleId) {
        return this.examples[exampleId] || null;
    }

    getSelectedExample() {
        return this.selectedExample ? this.examples[this.selectedExample] : null;
    }

    getAllExamples() {
        return this.examples;
    }

    getExamplesByCategory(category) {
        return Object.entries(this.examples).reduce((acc, [key, example]) => {
            if (example.category === category) {
                acc[key] = example;
            }
            return acc;
        }, {});
    }

    getCategories() {
        const categories = new Set();
        Object.values(this.examples).forEach(example => {
            if (example.category) {
                categories.add(example.category);
            }
        });
        return Array.from(categories);
    }
}
