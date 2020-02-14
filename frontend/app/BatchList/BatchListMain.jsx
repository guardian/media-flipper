import React from 'react';
import PropTypes from 'prop-types';
import MenuBanner from "../MenuBanner.jsx";
import css from "./BatchList.css";
import {Link} from 'react-router-dom';
import BatchListEntry from "./BatchListEntry.jsx";

class BatchListMain extends React.Component {
    constructor(props) {
        super(props);

        this.state = {
            loading: false,
            batches: [],
            lastError: null,
            pageSize: 50,
            onPage: 0
        };

        this.entryWasDeleted = this.entryWasDeleted.bind(this);
    }

    setStatePromise(newState) {
        return new Promise((resolve, reject)=>this.setState(newState, ()=>resolve()));
    }

    async loadBatches() {
        await this.setStatePromise({loading: true});

        const start = this.state.onPage * this.state.pageSize;
        const response = await fetch("/api/bulk/list?start=" + start + "&limit=" + this.state.pageSize);
        if(response.status===200) {
            const content = await response.json();
            return this.setStatePromise({loading: false, lastError: null, batches: content.entries})
        } else {
            const bodyText = await response.text();
            return this.setStatePromise({loading: false, lastError: bodyText});
        }
    }

    componentDidMount() {
        this.loadBatches();
    }

    entryWasDeleted(entryId) {
        const updatedBatches = this.state.batches.filter(entry=>entry.bulkListId!==entryId);
        this.setState({
            batches: updatedBatches
        })
    }

    render() {
        return <div>
            <MenuBanner/>
            <div className="batch-processing-grid">
                <h1 className="banner-header">Batch Processing</h1>
                <Link className="clickable button" style={{display: "inline", padding: "0.4em", textDecoration: "none"}} to="/batch/new">New batch...</Link>
                <ul className="batch-list">
                    {
                        this.state.batches.map(entry=><BatchListEntry entry={entry} entryWasDeleted={this.entryWasDeleted}/>)
                    }
                </ul>
            </div>
        </div>
    }
}

export default BatchListMain;