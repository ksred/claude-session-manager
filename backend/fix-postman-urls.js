#!/usr/bin/env node

// Simple script to fix Postman collection URLs by replacing {{baseUrl}} with absolute URLs

const fs = require('fs');
const path = require('path');

const COLLECTION_FILE = './postman_collection.json';
const OUTPUT_FILE = './postman_collection_fixed.json';
const BASE_URL = 'http://localhost:8080/api/v1';

function fixUrls(obj) {
    if (typeof obj !== 'object' || obj === null) {
        return obj;
    }
    
    if (Array.isArray(obj)) {
        return obj.map(fixUrls);
    }
    
    const result = {};
    for (const [key, value] of Object.entries(obj)) {
        if (key === 'url' && typeof value === 'object' && value.host) {
            // Fix Postman URL object
            result[key] = {
                ...value,
                raw: BASE_URL + '/' + (value.path || []).join('/') + 
                     (value.query && value.query.length > 0 ? '?' + value.query.map(q => q.key + '=' + (q.value || '')).join('&') : ''),
                protocol: 'http',
                host: ['localhost'],
                port: '8080',
                path: ['api', 'v1', ...(value.path || [])]
            };
        } else if (key === 'host' && Array.isArray(value) && value.includes('{{baseUrl}}')) {
            // Fix host arrays that contain {{baseUrl}}
            result[key] = ['localhost'];
        } else {
            result[key] = fixUrls(value);
        }
    }
    
    return result;
}

try {
    console.log('Reading Postman collection...');
    const collection = JSON.parse(fs.readFileSync(COLLECTION_FILE, 'utf8'));
    
    console.log('Fixing URLs...');
    const fixedCollection = fixUrls(collection);
    
    // Remove the baseUrl variable since we're using absolute URLs
    if (fixedCollection.variable) {
        fixedCollection.variable = fixedCollection.variable.filter(v => v.key !== 'baseUrl');
    }
    
    console.log('Writing fixed collection...');
    fs.writeFileSync(OUTPUT_FILE, JSON.stringify(fixedCollection, null, 2));
    
    console.log(`✅ Fixed Postman collection saved to: ${OUTPUT_FILE}`);
    console.log('Import this file into Postman - URLs will work immediately!');
    
} catch (error) {
    console.error('❌ Error:', error.message);
    process.exit(1);
}