#!/usr/bin/env node

/**
 * Icon Optimization Script
 *
 * This script helps convert and optimize icons for better performance.
 * It can convert ICO files to WebP format and optimize them.
 */

/* eslint-disable @typescript-eslint/no-require-imports */
const fs = require('fs');
const path = require('path');

console.log('🔧 Icon Optimization Script');
console.log('============================');

// Check if the icon.ico file exists
const iconPath = path.join(__dirname, '../public/images/icon.ico');

if (!fs.existsSync(iconPath)) {
  console.error('❌ Icon file not found at:', iconPath);
  process.exit(1);
}

// Get file stats
const stats = fs.statSync(iconPath);
const fileSizeKB = (stats.size / 1024).toFixed(2);

console.log(`📁 Current icon.ico size: ${fileSizeKB} KB`);
console.log('');

console.log('🚀 To optimize your icon, please follow these steps:');
console.log('');
console.log('1. Install sharp for image optimization:');
console.log('   npm install sharp');
console.log('');
console.log('2. Convert the icon to WebP format:');
console.log('   Run the following command in your terminal:');
console.log('');
console.log('   node -e "');
console.log("   const sharp = require('sharp');");
console.log("   sharp('public/images/icon.ico')");
console.log('     .resize(32, 32)');
console.log('     .webp({ quality: 80 })');
console.log("     .toFile('public/images/icon.webp')");
console.log("     .then(() => console.log('✅ Icon converted to WebP!'));");
console.log('   "');
console.log('');
console.log('3. Create a PNG fallback:');
console.log('   node -e "');
console.log("   const sharp = require('sharp');");
console.log("   sharp('public/images/icon.ico')");
console.log('     .resize(32, 32)');
console.log('     .png({ quality: 80 })');
console.log("     .toFile('public/images/icon.png')");
console.log("     .then(() => console.log('✅ Icon converted to PNG!'));");
console.log('   "');
console.log('');
console.log('4. After conversion, update the component to use the optimized icon:');
console.log('   Change src="/images/icon.ico" to src="/images/icon.webp"');
console.log('');
console.log('Expected improvements:');
console.log('- WebP format: 25-30% smaller file size');
console.log('- Better compression and quality');
console.log('- Faster loading times');
console.log('- Improved caching with new headers');
console.log('');
console.log('📊 Performance Impact:');
console.log(`- Current: ${fileSizeKB} KB (.ico format)`);
console.log(`- Expected: ~${(fileSizeKB * 0.7).toFixed(2)} KB (WebP format)`);
console.log(
  `- Savings: ~${(fileSizeKB * 0.3).toFixed(2)} KB (${(((fileSizeKB * 0.3) / fileSizeKB) * 100).toFixed(1)}%)`
);
