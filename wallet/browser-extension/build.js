import * as esbuild from 'esbuild';
import { NodeGlobalsPolyfillPlugin } from '@esbuild-plugins/node-globals-polyfill';
import { NodeModulesPolyfillPlugin } from '@esbuild-plugins/node-modules-polyfill';
import { minify } from 'html-minifier-terser';
import { readFileSync, writeFileSync, mkdirSync, copyFileSync, existsSync, readdirSync } from 'fs';
import { join } from 'path';

const isWatch = process.argv.includes('--watch');

// Build configuration
const buildOptions = {
  entryPoints: ['src/popup.js', 'src/background.js'],
  bundle: true,
  minify: true,
  sourcemap: false,
  target: ['chrome108', 'firefox108'],
  outdir: 'dist',
  format: 'esm',
  platform: 'browser',
  legalComments: 'none',
  treeShaking: true,
  plugins: [
    NodeGlobalsPolyfillPlugin({
      process: true,
      buffer: true,
    }),
    NodeModulesPolyfillPlugin(),
  ],
  define: {
    global: 'globalThis',
    'process.env.NODE_ENV': '"production"',
  },
};

async function build() {
  try {
    console.log('Building extension...');

    // Create dist directory
    if (!existsSync('dist')) {
      mkdirSync('dist', { recursive: true });
    }

    // Build JavaScript files
    await esbuild.build(buildOptions);
    console.log('✓ JavaScript files built');

    // Copy and minify HTML files
    const htmlFiles = ['popup.html'];
    for (const file of htmlFiles) {
      const html = readFileSync(file, 'utf8');
      const minified = await minify(html, {
        collapseWhitespace: true,
        removeComments: true,
        removeRedundantAttributes: true,
        removeScriptTypeAttributes: true,
        removeStyleLinkTypeAttributes: true,
        useShortDoctype: true,
        minifyCSS: true,
        minifyJS: true,
      });
      writeFileSync(join('dist', file), minified);
    }
    console.log('✓ HTML files minified and copied');

    // Copy CSS files
    if (existsSync('styles.css')) {
      copyFileSync('styles.css', 'dist/styles.css');
      console.log('✓ CSS files copied');
    }

    // Copy manifest
    copyFileSync('manifest.json', 'dist/manifest.json');
    console.log('✓ Manifest copied');

    // Copy icons
    if (existsSync('icons')) {
      if (!existsSync('dist/icons')) {
        mkdirSync('dist/icons', { recursive: true });
      }
      const icons = readdirSync('icons').filter(f => f.endsWith('.png'));
      for (const icon of icons) {
        copyFileSync(join('icons', icon), join('dist/icons', icon));
      }
      console.log('✓ Icons copied');
    }

    console.log('✓ Build complete!');
  } catch (error) {
    console.error('Build failed:', error);
    process.exit(1);
  }
}

if (isWatch) {
  console.log('Watching for changes...');
  const context = await esbuild.context(buildOptions);
  await context.watch();
} else {
  await build();
}
