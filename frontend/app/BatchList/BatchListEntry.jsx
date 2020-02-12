import React from 'react';
import PropTypes from 'prop-types';
import moment from 'moment';
import BatchStatusSummary from "./BatchStatusSummary.jsx";
import {FontAwesomeIcon} from "@fortawesome/react-fontawesome";
import {Link} from "react-router-dom";

class BatchListEntry extends React.Component {
    static propTypes = {
        entry: PropTypes.object.isRequired
    };

    constructor(props) {
        super(props);

        this.state = {
            loading: false,
            lastError: "",
            summaryData: null
        }
    }

    nameToDisplay() {
        if(this.props.entry.NickName && this.props.entry.NickName!==""){
            return <div><p>{this.props.entry.NickName}</p><p>{this.props.entry.bulkListId}</p></div>
        } else {
            return <p>{this.props.entry.bulkListId}</p>
        }
    }

    setStatePromise(newState) {
        return new Promise((resolve,reject)=>this.setState(newState, ()=>resolve()));
    }

    async loadSummaryData() {
        await this.setStatePromise({loading: true});
        const response = await fetch("/api/bulk/get?forId=" + this.props.entry.bulkListId);
        if(response.status===200){
            const content = await response.json();
            return this.setStatePromise({
                loading: false,
                lastError: null,
                summaryData: content
            });
        } else {
            const content = await response.text();
            try {
                const parsedErr = JSON.parse(content);
                return this.setStatePromise({
                    loading: false,
                    lastError: parsedErr.detail
                });
            } catch(e) {
                return this.setStatePromise({
                    loading: false,
                    lastError: content
                })
            }
        }
    }

    componentDidMount() {
        this.loadSummaryData();
    }

    render() {
        return <li className="batch-master-container" key={this.props.entry.bulkListId}>
            <div className="batch-entry-cell icon"><Link to={"/batch/" + this.props.entry.bulkListId} style={{color: "inherit"}}><FontAwesomeIcon icon="layer-group"/></Link></div>
            <div className="batch-entry-cell baseline">{this.nameToDisplay()}</div>
            <div className="batch-entry-cell mini">{moment(this.props.entry.creationTime).format("ddd, MMM Do YYYY, h:mm:ss a")}</div>
            <div className="batch-entry-cell baseline">{this.state.summaryData ? <BatchStatusSummary batchStatus={this.state.summaryData}/> : <p className="error-text">{this.state.lastError}</p>} </div>
        </li>
    }
}

export default BatchListEntry;