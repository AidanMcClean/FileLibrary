import React, { useState, useEffect } from 'react';
import axios from 'axios';
import { useNavigate } from 'react-router-dom';
import styles from './UploadForm.module.css';

function UploadForm() {
    const navigate = useNavigate();
    const [fileName, setFileName] = useState('');
    const [categories, setCategories] = useState([]);
    const [newCategory, setNewCategory] = useState('');
    const [selectedCategory, setSelectedCategory] = useState('');
    const [selectedFile, setSelectedFile] = useState(null);

    useEffect(() => {
        const fetchCategories = async () => {
            try {
                const response = await axios.get('http://localhost:8080/api/categories');
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

    const handleNameChange = (event) => {
        setFileName(event.target.value);
    };

    const handleCategoryChange = (event) => {
        setSelectedCategory(event.target.value);
    };

    const handleNewCategoryChange = (event) => {
        setNewCategory(event.target.value);
    };

    const addCategory = async () => {
        if (newCategory && !categories.some(category => category.name === newCategory)) {
            try {
                const response = await axios.post('http://localhost:8080/api/categories', {
                    name: newCategory
                });
                setCategories([...categories, response.data]);
                setSelectedCategory(response.data.id);
                setNewCategory('');
            } catch (error) {
                console.error('Error adding category:', error);
            }
        }
    };

    const handleSubmit = async (event) => {
        event.preventDefault();
        if (!selectedFile) {
            alert("Please select a file to upload.");
            return;
        }

        const formData = new FormData();
        formData.append('pdf', selectedFile);
        formData.append('category_id', selectedCategory);

        try {
            const response = await axios.post('http://localhost:8080/api/pdfs', formData, {
                headers: {
                    'Content-Type': 'multipart/form-data'
                }
            });
            console.log('File uploaded successfully', response.data);
            navigate('/');
        } catch (error) {
            console.error('Error uploading file:', error);
        }
    };

    return (
        <form onSubmit={handleSubmit} className={styles.form}>
            <input type="file" onChange={handleFileChange} accept=".pdf" className={styles.inputField} />
            <input type="text" value={fileName} onChange={handleNameChange} placeholder="Enter file name" className={styles.inputField} />
            <select value={selectedCategory} onChange={handleCategoryChange} className={styles.selectField}>
                <option value="">Select a category</option>
                {categories.map(category => (
                    <option key={category.id} value={category.id}>{category.name}</option>
                ))}
            </select>
            <input type="text" value={newCategory} onChange={handleNewCategoryChange} placeholder="Add new category" className={styles.inputField} />
            <button type="button" onClick={addCategory} className={styles.button}>Add Category</button>
            <button type="submit" className={styles.button}>Upload PDF</button>
        </form>
    );
}

export default UploadForm;
