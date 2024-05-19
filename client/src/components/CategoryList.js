import React, { useState, useEffect } from 'react';
import axios from 'axios';

function CategoryList() {
    const [categories, setCategories] = useState([]);
    const [pdfs, setPdfs] = useState([]);

    useEffect(() => {
        // fetch categories
        const fetchCategories = async () => {
            try {
                const categoryResponse = await axios.get('http://localhost:8080/api/categories');
                const pdfResponse = await axios.get('http://localhost:8080/api/pdfs');

                const pdfData = pdfResponse.data;
                const categoriesWithData = categoryResponse.data.filter(category => 
                    pdfData.some(pdf => pdf.category_id === category.id)
                );

                setCategories(categoriesWithData);
                setPdfs(pdfData);
            } catch (error) {
                console.error('Failed to fetch categories or PDFs:', error);
            }
        };

        fetchCategories();
    }, []);

    return (
        <div>
            {categories.map(category => (
                <div key={category.id}>
                    <h2>{category.name}</h2>
                    <ul>
                        {pdfs.filter(pdf => pdf.category_id === category.id).map(pdf => (
                            <li key={pdf.id}>{pdf.name}</li>
                        ))}
                    </ul>
                </div>
            ))}
        </div>
    );
}

export default CategoryList;
