import type {SidebarsConfig} from '@docusaurus/plugin-content-docs';

// This runs in Node.js - Don't use client-side code here (browser APIs, JSX...)

/**
 * Creating a sidebar enables you to:
 - create an ordered group of docs
 - render a sidebar for each doc of that group
 - provide next/previous navigation

 The sidebars can be generated from the filesystem, or explicitly defined here.

 Create as many sidebars as you want.
 */
const sidebars: SidebarsConfig = {
  gettingStartedSidebar: [
    'intro',
    {
      type: 'category',
      label: 'Getting Started',
      items: [
        'getting-started/installation',
        'getting-started/quick-start',
      ],
    },
  ],

  developersSidebar: [
    'developers/overview',
    {
      type: 'category',
      label: 'Developer Guides',
      items: [
        'developers/dex-integration',
        'developers/ibc-channels',
      ],
    },
  ],

  validatorsSidebar: [
    {
      type: 'category',
      label: 'Validators',
      items: [
        'validators/setup',
        'validators/monitoring',
      ],
    },
  ],
};

export default sidebars;
