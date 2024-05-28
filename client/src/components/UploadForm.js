import React, { useState, useEffect } from 'react';
import axios from 'axios';
import { useNavigate } from 'react-router-dom';
import styles from './UploadForm.module.css';

function UploadForm() {
    const navigate = useNavigate();
    const [displayName, setDisplayName] = useState('');
    const [categories, setCategories] = useState([]);
    const [selectedCategory, setSelectedCategory] = useState('');
    const [selectedFile, setSelectedFile] = useState(null);
    const [apiKey, setApiKey] = useState('');

    useEffect(() => {
        const fetchCategories = async () => {
            try {
                const response = await axios.get('https://aidanlibrarymanagementapp.azurewebsites.net/api/categories');
                setCategories(response.data);
            } catch (error) {
                console.error('Error fetching categories:', error);
            }
        };

        fetchCategories();
    }, []);

    const handleFileChange = (event) => {
        setSelectedFile(event.target.files[0]);
    };

    const handleDisplayNameChange = (event) => {
        setDisplayName(event.target.value);
    };

    const handleCategoryChange = (event) => {
        setSelectedCategory(event.target.value);
    };

    const handleApiKeyChange = (event) => {
        setApiKey(event.target.value);
    };

    const handleSubmit = async (event) => {
        event.preventDefault();
        if (!selectedFile) {
            alert("Please select a file to upload.");
            return;
        }

        if (!apiKey) {
            alert("Please enter the API key.");
            return;
        }

        const formData = new FormData();
        formData.append('pdf', selectedFile);
        formData.append('display_name', displayName);
        formData.append('category_id', selectedCategory);

        try {
            const response = await axios.post('https://aidanlibrarymanagementapp.azurewebsites.net/api/pdfs', formData, {
                headers: {
                    'Content-Type': 'multipart/form-data',
                    'x-api-key': apiKey
                }
            });
            console.log('File uploaded', response.data);
            navigate('/');
        } catch (error) {
            console.error('Error uploading file:', error);
        }
    };

    return (
        <form onSubmit={handleSubmit} className={styles.form}>
            <input type="file" onChange={handleFileChange} accept=".pdf" className={styles.inputField} />
            <input type="text" value={displayName} onChange={handleDisplayNameChange} placeholder="Enter display name" className={styles.inputField} />
            <select value={selectedCategory} onChange={handleCategoryChange} className={styles.selectField}>
                <option value="">Select a category</option>
                {categories.map(category => (
                    <option key={category.id} value={category.id}>{category.name}</option>
                ))}
            </select>
            <input type="password" value={apiKey} onChange={handleApiKeyChange} placeholder="Enter API key" className={styles.inputField} />
            <button type="submit" className={styles.button}>Upload PDF</button>
        </form>
    );
}

export default UploadForm;
